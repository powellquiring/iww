package iww

/*
Package iww creates list of resorces like all resources in the account or all resources in a group
- display the list (ls command)
- delete the list (rm command)

Eventually:
ls will output a list in terminal format with CRN as first word in each line
ls and rm will take a list of CRNs as input.  Handy for looking and then deleting

TODO shore up the names: resource, ServiceInstance, etc are all kind of similar but names are way different
*/

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/schollz/progressbar/v3"
)

type Key struct {
	region, instanceId string
}

const Verbose = true

func pbar(max int64, description ...string) *progressbar.ProgressBar {
	if Verbose {
		return progressbar.Default(max, description...)
	} else {
		return progressbar.DefaultSilent(max, description...)
	}
}

type ProgressBarWrapper struct {
	progressBar *progressbar.ProgressBar
	taken       int
	total       int
}

func NewProgressBarWrapper() *ProgressBarWrapper {
	const PBMAX = 1_000
	return &ProgressBarWrapper{progressBar: pbar(PBMAX), taken: 0, total: PBMAX}
}

// Make a new progress bar wrapper.  The size is a percent of total like progress
func (pbw *ProgressBarWrapper) subProgress(percent float64) *ProgressBarWrapper {
	pbmax := int((float64(pbw.total) * percent) + 0.5)
	if pbmax > pbw.total {
		pbmax = pbw.total - pbw.taken
	}
	return &ProgressBarWrapper{progressBar: pbw.progressBar, taken: 0, total: pbmax}
}

func (pbw *ProgressBarWrapper) progress(percent float64) {
	add := int((float64(pbw.total) * percent) + 0.5)
	if add+pbw.taken > pbw.total {
		add = pbw.total - pbw.taken
	}
	pbw.taken += add
	pbw.progressBar.Add(add)
}

// Global variables all the ones that end in Service are useful for operations
type Context struct {
	verboseLogger      *log.Logger
	progressBarWrapper *ProgressBarWrapper
	authenticator      core.Authenticator
	// apikey or token but not both, seems like authenticator would be enough but services like key protect
	// do not use an authenticator
	apikey            string
	token             string
	accountID         string
	region            string
	resourceGroupName string
	isType            bool   // only consider infrastructure services, vpc
	vpcid             string // only consider is resources that match the vpcid (isType must be true)
	resourceGroupID   string // initialized early can be trusted to be nil if no resource group provided
	crn               string // todo testing
	// the rest are initialized as needed and cached
	iamClient                  *iamidentityv1.IamIdentityV1
	IDToResourceGroupName      map[string]string
	nameToResourceGroupID      map[string]string
	nameToResourceGroupIDMutex sync.Mutex
	resourceManagerClient      *resourcemanagerv2.ResourceManagerV2
	resourceControllerClient   *resourcecontrollerv2.ResourceControllerV2
}

var GlobalContext *Context

// return the cached context or create it the first time called
func SetGlobalContext(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string, verbose bool) error {
	var err error
	if GlobalContext != nil {
		return nil
	}
	GlobalContext = &Context{}
	if !((apikey != "" && token == "") || (apikey == "" && token != "")) {
		return errors.New("one of apikey or token must be provided (not both)")
	}

	if verbose {
		GlobalContext.verboseLogger = log.New(os.Stdout, "-- ", log.LstdFlags)
		GlobalContext.verboseLogger.Print("start")
	} else {
		GlobalContext.verboseLogger = log.New(ioutil.Discard, "discarded", log.LstdFlags)
	}
	GlobalContext.progressBarWrapper = NewProgressBarWrapper()
	defer GlobalContext.progressBarWrapper.progress(0.10)
	GlobalContext.region = region
	GlobalContext.apikey = apikey
	GlobalContext.token = token
	if token != "" {
		GlobalContext.authenticator, err = core.NewBearerTokenAuthenticator(token)
		if err != nil {
			return err
		}
	} else {
		GlobalContext.authenticator = &core.IamAuthenticator{ApiKey: apikey}
	}

	if accountID == "" {
		if apikey != "" {
			iamClient, err := GlobalContext.getIamClient()
			if err != nil {
				return err
			}
			do := &iamidentityv1.GetAPIKeysDetailsOptions{
				IamAPIKey: &apikey,
			}
			apiKeyDetails, _, err := iamClient.GetAPIKeysDetails(do)
			if err != nil {
				return err
			}
			GlobalContext.accountID = *apiKeyDetails.AccountID
		}
	}
	GlobalContext.resourceGroupName = resourceGroupName
	GlobalContext.resourceGroupID = resourceGroupID
	GlobalContext.vpcid = vpcid
	if vpcid != "" {
		GlobalContext.isType = true
	}
	GlobalContext.resourceControllerClient, err = GlobalContext.getResourceControllerClient()
	if err != nil {
		return err
	}
	return SetGlobalContextResourceGroupID()
}

func SetGlobalContextResourceGroupID() error {
	if GlobalContext.resourceGroupID != "" {
		return nil // already have the ID
	}
	if GlobalContext.resourceGroupName == "" {
		return nil // no rg name nothing to do
	}
	if GlobalContext.accountID == "" {
		return errors.New("resource group name provided but without an account ID there is no way to get the group ID")
	}
	var err error
	GlobalContext.resourceGroupID, err = GlobalContext.getResourceGroup(GlobalContext.resourceGroupName)
	return err
}

func MustGlobalContext() *Context {
	if GlobalContext == nil {
		log.Fatal("GlobalContext must be initiliazed")
	}
	return GlobalContext
}

/*
Service instance state
State transition
start      -Fetch->   exists | deleted
exists     -Fetch->   exists
exists     -Fetch->   deleted
exists     -Destroy-> destroying
destroying -Fetch->   exists
destroying -Fetch->   destroying
destroying -Fetch->   deleted

See further definition of Fetch and Destroy below
*/
const (
	SIStateStart      = iota // filled with crn, no network activity
	SIStateExists            // cloud state has been fetched
	SIStateDestroying        // was in the exists state, a cloud call to delete was successful
	SIStateDeleted           // was in the destroying state, a cloud call to find it was successful but not found
)

func FormatInstance(name string, description string, crn Crn) string {
	return fmt.Sprint(crn.resourceType, " ", crn.vpcType, " ", name, " ", description, " ", crn.Crn)
}

type ResourceInstanceOperations interface {
	/*
	  Fetch function - get the resource from the cloud, if there is an error that indicartes a successful API call
	  with indication that the resource does not exist then resource changes to deleted (other
	  failures do not change the state of the resource)
	*/
	Fetch(si *ResourceInstanceWrapper) // fetch from cloud and upate the state, no need to retry in Fetch
	/*
	  Destroy - request a destroy of the resource.
	*/
	Destroy(si *ResourceInstanceWrapper) // fetch from cloud and upate the state, no need to retry in Fetch
	FormatInstance(si *ResourceInstanceWrapper, fast bool) string
}

// Crn is the parsed representation of a crn string. TODO make the vpc stuff more general if needed
type Crn struct {
	Crn          string
	resourceType string
	id           string
	vpcType      string
	vpcId        string
	region       string
	zone         string
}

func NewCrn(crn string) *Crn {
	//  0   1  2       3      4  5        6                                  78   9
	// "crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::vpc:r006-ea192ede-4e51-4126-b4f2-752912e92f72"
	//  0   1  2       3      4   5        6                                  7                                    89
	// "crn:v1:bluemix:public:kms:us-south:a/713c783d9a507a53135fe6793c37cc74:94f523f8-7e01-459d-a94d-89fd26f456e5::"
	parts := strings.Split(crn, ":")
	region := parts[5]
	zone := ""
	zoneDash := region[len(region)-2 : len(region)-1]
	if zoneDash == "-" {
		zone = region[len(region)-1:]
		if zone >= "0" && zone <= "9" {
			region = region[:len(region)-2]
		}
	}

	// todo refactor vpcType -> subType, vpcId -> subId
	return &Crn{
		Crn:          crn,
		resourceType: parts[4],
		id:           parts[7],
		vpcType:      parts[8],
		vpcId:        parts[9],
		region:       region,
		zone:         zone,
	}
}

// some resources do not have a real crn, so create what is needed, typically just a region and ID
func NewFakeCrn(resourceType, id, vpcType, vpcId, region string) *Crn {
	crn := "crn:v1:bluemix:public:" + resourceType + ":" + region + ":a/ACCOUNT:" + id + ":" + vpcType + ":" + vpcId
	return &Crn{
		Crn:          crn,
		resourceType: resourceType,
		id:           id,
		vpcType:      vpcType,
		vpcId:        vpcId,
		region:       region,
		zone:         "",
	}
}

func (crn *Crn) AsString() string {
	return crn.Crn
}

// wrappers are for both resourcecontrollerv2.ResourceInstance and ServiceInstance.
// Also contain state
type ResourceInstanceWrapper struct {
	operations ResourceInstanceOperations
	state      int
	crn        *Crn
	//context         *Context
	ResourceGroupID *string
	Name            *string
	resource        interface{}
}

func (ri *ResourceInstanceWrapper) Fetch() { ri.operations.Fetch(ri) }
func (ri *ResourceInstanceWrapper) FormatInstance(fast bool) string {
	return ri.operations.FormatInstance(ri, fast)
}
func (ri *ResourceInstanceWrapper) Destroy() { ri.operations.Destroy(ri) }

// ResourceGroup returns a string representation of the resource group.  Name if available
func (basic *ResourceInstanceWrapper) ResourceGroup() string {
	if resourceGroup, ok := MustGlobalContext().IDToResourceGroupName[*basic.ResourceGroupID]; ok {
		return resourceGroup
	} else {
		return *basic.ResourceGroupID
	}
}

// --------------------------------------
type TypicalServiceOperations struct {
	getResult   *resourcecontrollerv2.ResourceInstance
	getResponse *core.DetailedResponse
	getErr      error
}

func (s *TypicalServiceOperations) Destroy(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewDeleteResourceInstanceOptions(id)
	response, err := rc.DeleteResourceInstance(options)
	if err != nil {
		statusCode := "not_returned"
		if response != nil {
			statusCode = strconv.Itoa(response.StatusCode)
		}
		log.Print("TypicalServiceOpertions DeleteResourceInstance, StatusCode:", statusCode, " Crn:", si.crn.Crn, " err:", err.Error())
	}
}

func (s *TypicalServiceOperations) Fetch(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewGetResourceInstanceOptions(id)
	s.getResult, s.getResponse, s.getErr = rc.GetResourceInstance(options)
	if s.getErr != nil {
		if s.getResponse != nil && (s.getResponse.StatusCode == 404 || s.getResponse.StatusCode == 410) {
			si.state = SIStateDeleted
		} else {
			log.Print(s.getErr)
		}
	} else {
		si.state = SIStateExists
		if s.getResult != nil && *s.getResult.State == "removed" {
			si.state = SIStateDeleted
		}
	}
}

func (s *TypicalServiceOperations) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "-", *si.crn)
}

// --------------------------------------
// If the CRN can not be understood this unimplementedservice is used.
type UnimplementedServiceOperations struct {
}

func (s UnimplementedServiceOperations) Destroy(si *ResourceInstanceWrapper) {
	log.Print("Nil destroy should not have been called, crn:", si.crn.AsString())
}

func (s UnimplementedServiceOperations) Fetch(si *ResourceInstanceWrapper) {
	log.Print("Nil fetch crn:", si.crn.AsString())
	si.state = SIStateDeleted
}

func (s UnimplementedServiceOperations) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return "#-- " + si.crn.resourceType + " " + si.crn.resourceType + " " + si.crn.vpcType + " " + si.crn.Crn
	//  + FormatInstance(*si.Name, "NilServiceOpertions", *si.crn)
}

/* todo
//--------------------------------------
// ResourceToWrapper returns a resource from a resource instance
func operationsForWrappedResourceInstances(ri *ResourceInstanceWrapper) (ResourceInstanceOperations, error) {
	crn := ri.crn
	if crn.vpcType == "resource-key" {
		return &ResourceKeyOperations{}, nil
	}
	switch crn.resourceType {
	case "is":
		return NewVpcOperations(crn)
	case "logdna", "dns-svcs", "sysdig-monitor", "cloud-object-storage":
		return &TypicalServiceOperations{}, nil
	case "kms":
		return &KeyprotectServiceOpertions{}, nil
	case "transit":
		return &TransitGatewayServiceOpertions{}, nil
	default:
		return &TypicalServiceOperations{}, nil
		// return UnimplementedServiceOperations{}, nil
	}
}
*/

// pruneWrappedResourceInstancesByIs removes all non "is" resources from the list
func pruneWrappedResourceInstancesByIs(wrappedResourceInstances []*ResourceInstanceWrapper) []*ResourceInstanceWrapper {
	ret := make([]*ResourceInstanceWrapper, 0)
	for _, ri := range wrappedResourceInstances {
		crn := ri.crn
		if crn.resourceType == "is" {
			ret = append(ret, ri)
		}
	}
	return ret
}

func (context *Context) getIamClient() (client *iamidentityv1.IamIdentityV1, err error) {
	return iamidentityv1.NewIamIdentityV1UsingExternalConfig(&iamidentityv1.IamIdentityV1Options{
		Authenticator: context.authenticator,
	})
}

func (context *Context) getResourceManagerClient() (resourceManagerClient *resourcemanagerv2.ResourceManagerV2, err error) {
	return resourcemanagerv2.NewResourceManagerV2(&resourcemanagerv2.ResourceManagerV2Options{
		Authenticator: context.authenticator,
	})
}

func ApiEndpoint(documentedApiEndpoint string, region string) string {
	return strings.Replace(documentedApiEndpoint, "<region>", region, 1)
}

func (context *Context) getVpcClientFromRegion(region string) (service *vpcv1.VpcV1, err error) {
	return vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: MustGlobalContext().authenticator,
		URL:           ApiEndpoint("https://<region>.iaas.cloud.ibm.com/v1", region),
	})
}

func (context *Context) getVpcClient(crn *Crn) (service *vpcv1.VpcV1, err error) {
	region := crn.region
	return context.getVpcClientFromRegion(region)
}

func (context *Context) getTransitGatewayClient(crn *Crn) (*transitgatewayapisv1.TransitGatewayApisV1, error) {
	version := "2021-12-30"
	options := &transitgatewayapisv1.TransitGatewayApisV1Options{
		Version:       &version,
		Authenticator: context.authenticator,
		URL:           "https://transit.cloud.ibm.com/v1",
	}
	return transitgatewayapisv1.NewTransitGatewayApisV1(options)
	// todo
	// client.SetServiceURL("https://transit.cloud.ibm.com/v1")
}

// done with clients

func (context *Context) readResourceGroupsInitializeMaps() error {
	defer context.nameToResourceGroupIDMutex.Unlock()
	context.nameToResourceGroupIDMutex.Lock()
	if context.nameToResourceGroupID == nil {
		groupOptions := &resourcemanagerv2.ListResourceGroupsOptions{
			AccountID: &context.accountID,
		}
		resourceManagerClient, err := context.getResourceManagerClient()
		if err != nil {
			return err
		}
		returnListResourceGroups, _, err := resourceManagerClient.ListResourceGroups(groupOptions)
		if err != nil {
			return err
		}
		context.nameToResourceGroupID = makeNameToResourceGroupID(returnListResourceGroups)
		context.IDToResourceGroupName = makeIdToResourceGroup(context.nameToResourceGroupID)
	}
	return nil
}

// todo change name go getResourcGroupID
func (context *Context) getResourceGroup(resourceGroupName string) (string, error) {
	err := context.readResourceGroupsInitializeMaps()
	if err != nil {
		return "", err
	}
	resourceGroup, found := context.nameToResourceGroupID[resourceGroupName]
	if found {
		return resourceGroup, nil
	} else {
		return "", errors.New("resource group not found, name: " + resourceGroupName)
	}
}

// ------------------------------------
// Global variable initialization section
func (context *Context) getResourceGroupName(id string, fast bool) string {
	if context.resourceGroupID == id {
		return context.resourceGroupName
	}
	if context.IDToResourceGroupName == nil {
		if fast {
			return ""
		}
		if err := context.readResourceGroupsInitializeMaps(); err != nil {
			log.Print(err)
			return ""
		}
	}
	if context.IDToResourceGroupName == nil {
		return ""
	}
	if rg, ok := context.IDToResourceGroupName[id]; ok {
		return rg
	}
	return ""
}

func makeNameToResourceGroupID(returnListResourceGroups *resourcemanagerv2.ResourceGroupList) map[string]string {
	nameToResourceGroupID := make(map[string]string, 0)
	for _, resourceGroup := range returnListResourceGroups.Resources {
		nameToResourceGroupID[*resourceGroup.Name] = *resourceGroup.ID
	}
	return nameToResourceGroupID
}

func makeIdToResourceGroup(nameToResourceGroupID map[string]string) map[string]string {
	IdToResourceGroup := make(map[string]string, 0)
	for resourceGroupName, resourceGroupID := range nameToResourceGroupID {
		IdToResourceGroup[resourceGroupID] = resourceGroupName
	}
	return IdToResourceGroup
}

func exit() {
	os.Exit(1)
}

func printErrExit(err error) {
	if err != nil {
		fmt.Println(err)
		exit()
	}
}

func newInt64(i int64) *int64 {
	return &i
}

// ListExpandFastPruneAddOperations is the list from the RC, expanded to include extra instances not in RC
// then fast prune (no fetching instances) then add operations
type ResourceFinder interface {
	// Find will take the current resource instances and find more and adjust the operators
	Find([]*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error)
}

// resourceFinders is a squential list of finders, order is important since most finders expect
// to be passed a list of resource from the resource controller, RC
var resourceFinders []ResourceFinder = []ResourceFinder{
	ResourceFinderRC{},
	ResourceFinderSchematics{},
	ResourceFinderTransitGateway{},
	ResourceFinderVpc{},
	ResourceFinderResourceKeys{},
	ResourceFinderDns{},
	ResourceFinderKeyProtect{},
}

// Return the resources in the cloud, if no filters then all of them, see filtering
func ListExpandFastPruneAddOperations() ([]*ResourceInstanceWrapper, error) {
	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	percent := 1.0 / float64(len(resourceFinders))
	pbw := MustGlobalContext().progressBarWrapper.subProgress(100.0)
	for _, finder := range resourceFinders {
		var err error
		wrappedResourceInstances, err = finder.Find(wrappedResourceInstances)
		if err != nil {
			return nil, err
		}
		pbw.progress(percent)
	}
	context := MustGlobalContext()
	if context.isType {
		wrappedResourceInstances = pruneWrappedResourceInstancesByIs(wrappedResourceInstances)
	}

	// the original resource controller list only included resources from the resource group
	// but the finders could have added resources in the wrong group.
	if context.resourceGroupID != "" {
		var ret []*ResourceInstanceWrapper
		for _, ri := range wrappedResourceInstances {
			if *ri.ResourceGroupID == context.resourceGroupID {
				ret = append(ret, ri)
			}
		}
		return ret, nil
	}
	return wrappedResourceInstances, nil
}

const Async = true

func fetchStoreResults(ri *ResourceInstanceWrapper, wg *sync.WaitGroup) {
	defer wg.Done()
	ri.Fetch()
}

// List is called from all commands (rm, ls, tst) to to find the list of resources that match the global context.
// important the the set of resources for ls and rm are the same for good user experience
// if fast do not fetch the instances
func List(fast bool) ([]*ResourceInstanceWrapper, error) {
	context := MustGlobalContext()
	wrappedResourceInstances, err := ListExpandFastPruneAddOperations()
	if err != nil {
		return nil, err
	}

	if fast {
		return wrappedResourceInstances, nil
	} else {
		fetchedRis := make([]*ResourceInstanceWrapper, len(wrappedResourceInstances))
		var wg sync.WaitGroup
		// for some filtering, like vpcid, it is required to fetch.  To be consistent fetch now
		for index, ri := range wrappedResourceInstances {
			fetchedRis[index] = ri
			// context.progressBarWrapper.progress(0.0)
			wg.Add(1)
			time.Sleep(10 * time.Millisecond) // avoid rate limiting
			if Async {
				go fetchStoreResults(ri, &wg)
			} else {
				fetchStoreResults(ri, &wg)
			}
		}
		wg.Wait()
		ret := make([]*ResourceInstanceWrapper, 0)
		for _, ri := range fetchedRis {
			if context.vpcid != "" {
				// filter based on vpcid
				if vpcOperations, ok := ri.operations.(VpcResourceInstanceOperations); ok {
					if context.vpcid == vpcOperations.Vpcid() {
						ret = append(ret, ri)
					}
				}
			} else {
				ret = append(ret, ri)
			}
		}
		return ret, nil
	}
}

func NewResourceInstanceWrapper(crn *Crn, resourceGroupID *string, name *string) *ResourceInstanceWrapper {
	return &ResourceInstanceWrapper{
		crn:             crn,
		ResourceGroupID: resourceGroupID,
		Name:            name,
	}
}

// NewSubInstance creates a resource for a subtype of a parent type.  The "iww-" is added to the subtype to identify it as
// an iww subtype and not an actual one
func NewSubInstance(parent *ResourceInstanceWrapper, subType, id string, name *string, operations ResourceInstanceOperations) *ResourceInstanceWrapper {
	typeCrn := parent.crn.Crn
	crnString := typeCrn[0:len(typeCrn)-1] + "iww-" + subType + ":" + id
	crn := NewCrn(crnString)
	ret := NewResourceInstanceWrapper(crn, parent.ResourceGroupID, name)
	// zone.resource = dz
	ret.operations = operations
	return ret
}

// ls with apikey from iww command line
func Ls(apikey, region string, resourceGroupName string, vpcid string, fast bool, verbose bool) error {
	return LsCommon(apikey, "", "", region, resourceGroupName, "", vpcid, fast, verbose)
}

// ls with context manager from ibmcloud cli
func LsWithToken(token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string, fast bool, verbose bool) error {
	return LsCommon("", token, accountID, region, resourceGroupName, resourceGroupID, vpcid, fast, verbose)
}

func LsCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string, fast bool, verbose bool) error {
	if vpcid != "" {
		if fast {
			return errors.New("fast and vpcid are not compatible")
		}

	}
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID, vpcid, verbose); err != nil {
		return err
	}
	wrappedResourceInstances, err := List(fast)
	if err != nil {
		return err
	}
	return lsOutput(wrappedResourceInstances, fast)
}

func lsOutput(wrappedResourceInstances []*ResourceInstanceWrapper, fast bool) error {
	context := MustGlobalContext()
	unimplementedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	missingResourceInstances := make([]*ResourceInstanceWrapper, 0)
	existingResourceInstances := make([]*ResourceInstanceWrapper, 0)
	if fast {
		existingResourceInstances = append(existingResourceInstances, wrappedResourceInstances...)
	} else {
		// fetch all of the resources
		for _, ri := range wrappedResourceInstances {
			if _, ok := ri.operations.(UnimplementedServiceOperations); ok {
				unimplementedResourceInstances = append(unimplementedResourceInstances, ri)
			} else {
				if ri.state == SIStateDeleted {
					missingResourceInstances = append(missingResourceInstances, ri)
				} else {
					existingResourceInstances = append(existingResourceInstances, ri)
				}
			}
		}
	}
	if len(unimplementedResourceInstances) > 0 {
		fmt.Println("#Unimplemented resource instances")
		PrintResourceInstances(context, fast, unimplementedResourceInstances)
	}
	if len(missingResourceInstances) > 0 {
		fmt.Println("#Missing resource instances")
		PrintResourceInstances(context, fast, missingResourceInstances)
	}
	fmt.Println("#Resource instances")
	PrintResourceInstances(context, fast, existingResourceInstances)
	return nil
}

type RIWs []*ResourceInstanceWrapper

func (ris RIWs) Len() int           { return len(ris) }
func (ris RIWs) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }
func (ris RIWs) Less(i, j int) bool { return ris[i].crn.Crn < ris[j].crn.Crn }

func PrintResourceInstances(context *Context, fast bool, wrappedResourceInstances []*ResourceInstanceWrapper) {
	// Sort the instance by resource group
	// byResourceGroup := make(map[string][]*ResourceInstanceWrapper)
	byResourceGroup := make(map[string]RIWs)
	groupIds := make([]string, 0)
	for _, ri := range wrappedResourceInstances {
		groupId := *ri.ResourceGroupID
		if _, ok := byResourceGroup[groupId]; !ok {
			// byResourceGroup[groupId] = make([]*ResourceInstanceWrapper, 0)
			groupIds = append(groupIds, groupId)
			byResourceGroup[groupId] = make(RIWs, 0)
		}
		byResourceGroup[groupId] = append(byResourceGroup[groupId], ri)
	}
	sort.Strings(groupIds)
	for _, groupId := range groupIds {
		// for groupId, ris := range byResourceGroup {
		ris := byResourceGroup[groupId]
		sort.Sort(ris)

		fmt.Println("#", groupId, "(", context.getResourceGroupName(groupId, fast), ")")
		for _, ri := range ris {
			fmt.Println(ri.FormatInstance(fast))
		}
	}
}

/*
State transition
start      -fetch->   exists | deleted
exists     -fetch->   exists
exists     -fetch->   deleted
exists     -destroy-> destroying
destroying -fetch->   exists
destroying -fetch->   destroying
destroying -fetch->   deleted
*/
func RmServiceInstances(serviceInstances []*ResourceInstanceWrapper) error {
	nextServiceInstances := make([]*ResourceInstanceWrapper, 0)
	for i := 0; i < 100 && len(serviceInstances) > 0; i++ {
		for _, si := range serviceInstances {
			switch si.state {
			case SIStateStart:
				fmt.Println("start:", si.FormatInstance(true))
				nextServiceInstances = append(nextServiceInstances, si)
			case SIStateExists:
				fmt.Println("destroying", si.FormatInstance(true))
				si.Destroy()
				nextServiceInstances = append(nextServiceInstances, si)
			case SIStateDestroying:
				fmt.Println("waiting", si.FormatInstance(true))
				nextServiceInstances = append(nextServiceInstances, si)
			case SIStateDeleted:
				fmt.Println("deleted:", si.FormatInstance(true))
				// making some progress
				i = 0
				//nextServiceInstances = append(nextServiceInstances, si)
			}
			si.Fetch()
		}
		serviceInstances = pruneResourcesThatDoNotExist(nextServiceInstances)
		nextServiceInstances = make([]*ResourceInstanceWrapper, 0)
		time.Sleep(2 * time.Second)
	}
	if len(nextServiceInstances) != 0 {
		return errors.New("some service instances not deleted")
	}
	return nil
}

// prune out the resources that are no longer in the resource controller
func pruneResourcesThatDoNotExist(nextServiceInstances []*ResourceInstanceWrapper) []*ResourceInstanceWrapper {
	resources, err := ListExpandFastPruneAddOperations() // assume if they are not in the RC they can be pruned
	if err != nil {
		log.Print("can not prune resources, err:", err)
		return nextServiceInstances
	}
	// map from crn to resource
	crnToResource := make(map[string]*ResourceInstanceWrapper, 0)
	for _, ri := range resources {
		crn := ri.crn.Crn
		if _, ok := crnToResource[crn]; ok {
			log.Print("multiple resources have the same crn:", crn)
		} else {
			crnToResource[crn] = ri
		}
	}

	// only return resources that are in the map
	ret := make([]*ResourceInstanceWrapper, 0)
	for _, ri := range nextServiceInstances {
		if _, ok := crnToResource[ri.crn.Crn]; ok {
			ret = append(ret, ri)
		}
	}
	return ret
}

func Rm(apikey, region string, resourceGroupName string, fileName string, vpcid string, crn string, force bool, verbose bool) error {
	if fileName != "" {
		log.Print("rm from file not supported, fileName:", fileName)
		return nil
	}
	return RmCommon(apikey, "", "", region, resourceGroupName, "", vpcid, crn, force, verbose)
}

func RmWithToken(token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string, crn string, force bool, verbose bool) error {
	return RmCommon("", token, accountID, region, resourceGroupName, resourceGroupID, vpcid, crn, force, verbose)
}

func RmCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string, crn string, force bool, verbose bool) error {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID, vpcid, verbose); err != nil { // todo
		return err
	}
	if resourceGroupName == "" && resourceGroupID == "" && vpcid == "" && crn == "" {
		fmt.Print("Removing all resources not currently supported, select a resource group, vpc or crn")
		return nil
	}
	serviceInstances, err := List(false)
	if err != nil {
		return err
	}

	var crnSi *ResourceInstanceWrapper
	if crn != "" {
		for _, si := range serviceInstances {
			if si.crn.Crn == crn {
				crnSi = si
				serviceInstances = []*ResourceInstanceWrapper{crnSi}
			}
		}
		if crnSi == nil {
			fmt.Println("crn not found, crn:", crn)
			return nil
		}
	}

	lsOutput(serviceInstances, false)
	if !force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Remove these resources? Y/n: ")
		text, _ := reader.ReadString('\n')
		text = strings.ToLower(strings.TrimSpace(text))
		fmt.Println(text)
		force = len(text) == 0 || strings.HasPrefix(text, "y")
	}

	if !force {
		return nil
	}

	RmServiceInstances(serviceInstances)
	return nil
}

func Tst(apikey, region string, resourceGroupName string) error {
	return TstCommon(apikey, "", "", region, resourceGroupName, "")
}

func TstCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string) error {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID, "", true); err != nil { // todo
		return err
	}
	serviceInstances, err := List(false)
	if err != nil {
		return err
	}
	TstServiceInstances(serviceInstances)
	return nil
}
func TstServiceInstances(serviceInstances []*ResourceInstanceWrapper) error {
	nextServiceInstances := make([]*ResourceInstanceWrapper, 0)
	for _, si := range serviceInstances {
		switch si.state {
		case SIStateStart:
			fmt.Println("start:", si.FormatInstance(true))
			nextServiceInstances = append(nextServiceInstances, si)
		case SIStateExists:
			fmt.Println("destroying", si.FormatInstance(true))
			si.Destroy()
			nextServiceInstances = append(nextServiceInstances, si)
		case SIStateDestroying:
			fmt.Println("waiting", si.FormatInstance(true))
			nextServiceInstances = append(nextServiceInstances, si)
		case SIStateDeleted:
			fmt.Println("deleted:", si.FormatInstance(true))
		}
		si.Fetch()
		serviceInstances = pruneResourcesThatDoNotExist(nextServiceInstances)
		nextServiceInstances = make([]*ResourceInstanceWrapper, 0)
		time.Sleep(2 * time.Second)
	}
	if len(nextServiceInstances) != 0 {
		return errors.New("some service instances not deleted")
	}
	return nil
}

func Tag(apikey, resourceGroup string) error {
	//fmt.Println("tag", apikey, resourceGroup)
	return nil
}
