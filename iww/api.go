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
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	kp "github.com/IBM/keyprotect-go-client"
	"github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/IBM/platform-services-go-sdk/iamidentityv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

type Key struct {
	region, instanceId string
}

// Global variables all the ones that end in Service are useful for operations
type Context struct {
	authenticator core.Authenticator
	// apikey or token but not both, seems like authenticator would be enough but services like key protect
	// do not use an authenticator
	apikey            string
	token             string
	accountID         string
	region            string
	resourceGroupName string
	resourceGroupID   string // initialized early can be trusted to be nil if no resource group provided
	// the rest are initialized as needed and cached
	iamClient                *iamidentityv1.IamIdentityV1
	nameToResourceGroupID    map[string]string
	IDToResourceGroupName    map[string]string
	resourceManagerClient    *resourcemanagerv2.ResourceManagerV2
	resourceControllerClient *resourcecontrollerv2.ResourceControllerV2
	KeyProtectClients        map[Key]*kp.Client
	TransitGatewayClient     *transitgatewayapisv1.TransitGatewayApisV1
	VpcClients               map[ /*region*/ string]*vpcv1.VpcV1
}

var GlobalContext *Context

// return the cached context or create it the first time called
func SetGlobalContext(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string) error {
	if GlobalContext != nil {
		return nil
	}
	if !((apikey != "" && token == "") || (apikey == "" && token != "")) {
		return errors.New("one of apikey or token must be provided (not both)")
	}
	var err error
	GlobalContext = &Context{}
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
	// resource   resourcecontrollerv2.ResourceInstance
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

//--------------------------------------
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

//--------------------------------------
type ResourceKeyOperations struct {
	getResult   *resourcecontrollerv2.ResourceKey
	getResponse *core.DetailedResponse
	getErr      error
}

func (s *ResourceKeyOperations) Destroy(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewDeleteResourceKeyOptions(id)
	response, err := rc.DeleteResourceKey(options)
	if err != nil {
		statusCode := "not_returned"
		if response != nil {
			statusCode = strconv.Itoa(response.StatusCode)
		}
		log.Print("TypicalServiceOpertions DeleteResourceKey, StatusCode:", statusCode, " Crn:", si.crn.Crn, " err:", err.Error())
	}
}

func (s *ResourceKeyOperations) Fetch(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewGetResourceKeyOptions(id)
	s.getResult, s.getResponse, s.getErr = rc.GetResourceKey(options)
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

func (s *ResourceKeyOperations) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "-", *si.crn)
}

//--------------------------------------
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

//--------------------------------------
type KeyprotectServiceOpertions struct {
}

func (s *KeyprotectServiceOpertions) Destroy(si *ResourceInstanceWrapper) {
	crn := si.crn
	if client, err := MustGlobalContext().getKeyProtectClient(crn); err == nil {
		pageSize := 3
		keys := make([]kp.Key, 0)
		// 100 times through max, avoid infinite loop
		for i := 0; i < 100; i = i + 1 {
			getKeys, err := client.GetKeys(context.Background(), pageSize, i*pageSize)
			if err == nil {
				keys = append(keys, getKeys.Keys...)
				if len(getKeys.Keys) < pageSize {
					break
				}
			} else {
				log.Println("KeyprotectServiceOpertions GetKeys failed err:", err)
				break
			}
		}
		for _, key := range keys {
			delKey, err := client.DeleteKey(context.Background(), key.ID, kp.ReturnRepresentation, kp.ForceOpt{Force: true})
			if err != nil {
				log.Print("KeyprotectServiceOpertions error while deleting the key: ", err)
			} else {
				log.Print("KeyprotectServiceOpertions deleted key name:", delKey.Name, ", id:", delKey.ID)
			}
		}
	}
	(&TypicalServiceOperations{}).Destroy(si)
}

// kms removes itsef from the resource controller but continues to return a state of removed
func (s *KeyprotectServiceOpertions) Fetch(si *ResourceInstanceWrapper) {
	(&TypicalServiceOperations{}).Fetch(si)
}

func (s *KeyprotectServiceOpertions) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "-", *si.crn)
}

//--------------------------------------
type TransitGatewayServiceOpertions struct {
}

func (s *TransitGatewayServiceOpertions) Destroy(si *ResourceInstanceWrapper) {
	crn := si.crn
	if client, err := MustGlobalContext().getTransitGatewayClient(crn); err == nil {
		deleteTransitGatewayOptions := client.NewDeleteTransitGatewayOptions(
			crn.vpcId,
			// crn.Crn,
		)

		response, err := client.DeleteTransitGateway(deleteTransitGatewayOptions)
		if err != nil {
			statusCode := 0
			if response != nil {
				statusCode = response.StatusCode
				return
			}
			log.Print("TransitGateway error while deleting status code:", statusCode, ", err:", err)
		}
	} else {
		log.Print("Error deleting transit gateway err:", err)
	}
}

func (s *TransitGatewayServiceOpertions) Fetch(si *ResourceInstanceWrapper) {
	(&TypicalServiceOperations{}).Fetch(si)
}

func (s *TransitGatewayServiceOpertions) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return (&TypicalServiceOperations{}).FormatInstance(si, fast)
}

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

//------------------------------------
// Global variable initialization section
func (context *Context) getResourceControllerClient() (client *resourcecontrollerv2.ResourceControllerV2, err error) {
	if context.resourceControllerClient != nil {
		client = context.resourceControllerClient
	} else {
		if client, err = resourcecontrollerv2.NewResourceControllerV2(&resourcecontrollerv2.ResourceControllerV2Options{
			Authenticator: context.authenticator,
		}); err == nil {
			context.resourceControllerClient = client
		}
	}
	return
}

func (context *Context) getIamClient() (client *iamidentityv1.IamIdentityV1, err error) {
	if context.iamClient != nil {
		client = context.iamClient
	} else {
		if client, err = iamidentityv1.NewIamIdentityV1UsingExternalConfig(&iamidentityv1.IamIdentityV1Options{
			Authenticator: context.authenticator,
		}); err == nil {
			context.iamClient = client
		}
	}
	return
}

func (context *Context) getResourceManagerClient() (resourceManagerClient *resourcemanagerv2.ResourceManagerV2, err error) {
	if context.resourceManagerClient != nil {
		resourceManagerClient = context.resourceManagerClient
	} else {
		if resourceManagerClient, err = resourcemanagerv2.NewResourceManagerV2(&resourcemanagerv2.ResourceManagerV2Options{
			Authenticator: context.authenticator,
		}); err == nil {
			context.resourceManagerClient = resourceManagerClient
		}
	}
	return
}

func ApiEndpoint(documentedApiEndpoint string, region string) string {
	return strings.Replace(documentedApiEndpoint, "<region>", region, 1)
}

func (context *Context) getKeyProtectClient(crn *Crn) (*kp.Client, error) {
	region := crn.region
	instanceId := crn.Crn
	key := Key{region, instanceId}
	if context.KeyProtectClients == nil {
		context.KeyProtectClients = make(map[Key]*kp.Client, 0)
	}
	if client, ok := context.KeyProtectClients[key]; ok {
		return client, nil
	} else {
		config := kp.ClientConfig{
			BaseURL:       ApiEndpoint("https://<region>.kms.cloud.ibm.com", region),
			APIKey:        context.apikey,
			Authorization: context.token,
			TokenURL:      kp.DefaultTokenURL,
			InstanceID:    crn.id,
			Verbose:       kp.VerboseFailOnly,
		}
		if client, err := kp.New(config, kp.DefaultTransport()); err == nil {
			context.KeyProtectClients[key] = client
			return client, nil
		} else {
			return nil, err
		}
	}
}

func (context *Context) getVpcClient(crn *Crn) (service *vpcv1.VpcV1, err error) {
	region := crn.region
	if context.VpcClients == nil {
		context.VpcClients = make(map[string]*vpcv1.VpcV1, 0)
	}
	if client, ok := context.VpcClients[region]; ok {
		return client, nil
	} else {
		if client, err = vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: MustGlobalContext().authenticator,
			URL:           ApiEndpoint("https://<region>.iaas.cloud.ibm.com/v1", region),
		}); err == nil {
			context.VpcClients[region] = client
			return client, nil
		} else {
			return nil, err
		}
	}
}

func (context *Context) getTransitGatewayClient(crn *Crn) (*transitgatewayapisv1.TransitGatewayApisV1, error) {
	if context.TransitGatewayClient != nil {
		return context.TransitGatewayClient, nil
	}
	version := "2021-12-30"
	options := &transitgatewayapisv1.TransitGatewayApisV1Options{
		Version:       &version,
		Authenticator: context.authenticator,
	}
	client, err := transitgatewayapisv1.NewTransitGatewayApisV1(options)
	if err != nil {
		return client, err
	}
	client.SetServiceURL("https://transit.cloud.ibm.com/v1")
	context.TransitGatewayClient = client
	return context.TransitGatewayClient, err
}

func (context *Context) readResourceGroupsInitializeMaps() error {
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

//------------------------------------
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

// Return the list of resource instances matching the option
func ResourceInstances(service *resourcecontrollerv2.ResourceControllerV2, lrio *resourcecontrollerv2.ListResourceInstancesOptions) ([]resourcecontrollerv2.ResourceInstance, error) {
	resourceInstances := make([]resourcecontrollerv2.ResourceInstance, 0)
	// limit the number of calls
	for i := 0; i < 100; i++ {
		resourceInstancesList, _, err := service.ListResourceInstances(lrio)
		if err != nil {
			return resourceInstances, err
		}
		resourceInstances = append(resourceInstances, resourceInstancesList.Resources...)
		if resourceInstancesList.NextURL == nil {
			break // yeah, got them all
		}
		startString, err := core.GetQueryParam(resourceInstancesList.NextURL, "start")
		if err != nil {
			return resourceInstances, err
		}
		lrio.SetStart(*startString)
	}
	return resourceInstances, nil
}

func List() ([]*ResourceInstanceWrapper, error) {
	resourceControllerClient, err := MustGlobalContext().getResourceControllerClient()
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)

	resourceInstances, err := readResourceInstances(resourceControllerClient)
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances = append(wrappedResourceInstances, resourceInstances...)

	serviceKeys, err := readResourceKeys(resourceControllerClient)
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances = append(wrappedResourceInstances, serviceKeys...)

	vpcExtraInstances, err := readVpcExtraInstances(wrappedResourceInstances)
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances = append(wrappedResourceInstances, vpcExtraInstances...)

	for _, ri := range wrappedResourceInstances {
		operations, err := operationsForWrappedResourceInstances(ri)
		if err != nil {
			return nil, err
		}
		ri.operations = operations
	}
	return wrappedResourceInstances, nil
}

func NewResourceInstanceWrapper(crn *Crn, resourceGroupID *string, name *string) *ResourceInstanceWrapper {
	return &ResourceInstanceWrapper{
		crn:             crn,
		ResourceGroupID: resourceGroupID,
		Name:            name,
	}
}

func readResourceInstances(resourceControllerClient *resourcecontrollerv2.ResourceControllerV2) ([]*ResourceInstanceWrapper, error) {
	context := MustGlobalContext()
	lriOptions := resourceControllerClient.NewListResourceInstancesOptions()
	if context.resourceGroupID != "" {
		lriOptions.SetResourceGroupID(context.resourceGroupID)
	}
	resourceInstances, err := ResourceInstances(resourceControllerClient, lriOptions)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	var lastErr error
	for _, ri := range resourceInstances {
		crn := NewCrn(*ri.CRN)
		si := NewResourceInstanceWrapper(crn, ri.ResourceGroupID, ri.Name)
		if err != nil {
			lastErr = err
			fmt.Println("BAD CRN:", *ri.CRN)
		} else {
			if context.region == "" || context.region == crn.region {
				wrappedResourceInstances = append(wrappedResourceInstances, si)
			}
		}
	}
	return wrappedResourceInstances, lastErr
}

// Return the list of service keys matching the option
func ResourceKeys(service *resourcecontrollerv2.ResourceControllerV2, lrio *resourcecontrollerv2.ListResourceKeysOptions) ([]resourcecontrollerv2.ResourceKey, error) {
	resourceKeys := make([]resourcecontrollerv2.ResourceKey, 0)
	// limit the number of calls
	for i := 0; i < 100; i++ {
		resourceInstancesList, _, err := service.ListResourceKeys(lrio)
		if err != nil {
			return resourceKeys, err
		}
		resourceKeys = append(resourceKeys, resourceInstancesList.Resources...)
		if resourceInstancesList.NextURL == nil {
			break // yeah, got them all
		}
		startString, err := core.GetQueryParam(resourceInstancesList.NextURL, "start")
		if err != nil {
			return resourceKeys, err
		}
		lrio.SetStart(*startString)
	}
	return resourceKeys, nil
}

func readResourceKeys(resourceControllerClient *resourcecontrollerv2.ResourceControllerV2) ([]*ResourceInstanceWrapper, error) {
	context := MustGlobalContext()
	lrkOptions := resourceControllerClient.NewListResourceKeysOptions()
	if context.resourceGroupID != "" {
		lrkOptions.SetResourceGroupID(context.resourceGroupID)
	}
	resourceKeys, err := ResourceKeys(resourceControllerClient, lrkOptions)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	var lastErr error
	for _, ri := range resourceKeys {
		crn := NewCrn(*ri.CRN)
		si := NewResourceInstanceWrapper(crn, ri.ResourceGroupID, ri.Name)
		if err != nil {
			lastErr = err
			fmt.Println("BAD CRN:", *ri.CRN)
		} else {
			if context.region == "" || context.region == crn.region {
				wrappedResourceInstances = append(wrappedResourceInstances, si)
			}
		}
	}
	return wrappedResourceInstances, lastErr
}

// ls with apikey from iww command line
func Ls(apikey, region string, resourceGroupName string, fast bool) error {
	return LsCommon(apikey, "", "", region, resourceGroupName, "", fast)
}

// ls with context manager from ibmcloud cli
func LsWithToken(token string, accountID string, region string, resourceGroupName string, resourceGroupID string, fast bool) error {
	return LsCommon("", token, accountID, region, resourceGroupName, resourceGroupID, fast)
}

func LsCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string, fast bool) error {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID); err != nil {
		return err
	}
	wrappedResourceInstances, err := List()
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
				ri.Fetch()
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

func PrintResourceInstances(context *Context, fast bool, wrappedResourceInstances []*ResourceInstanceWrapper) {
	// Sort the instance by resource group
	byResourceGroup := make(map[string][]*ResourceInstanceWrapper)
	for _, ri := range wrappedResourceInstances {
		groupId := *ri.ResourceGroupID
		if _, ok := byResourceGroup[groupId]; !ok {
			byResourceGroup[groupId] = make([]*ResourceInstanceWrapper, 0)
		}
		byResourceGroup[groupId] = append(byResourceGroup[groupId], ri)
	}
	for groupId, ris := range byResourceGroup {
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
	resources, err := List()
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

func Rm(apikey, region string, resourceGroupName string, fileName string) error {
	if fileName != "" {
		log.Print("rm from file not suppor, fileName:", fileName)
		return nil
	}
	return RmCommon(apikey, "", "", region, resourceGroupName, "")
}

func RmWithToken(token string, accountID string, region string, resourceGroupName string, resourceGroupID string) error {
	return RmCommon("", token, accountID, region, resourceGroupName, resourceGroupID)
}

func RmCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string) error {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID); err != nil {
		return err
	}
	if resourceGroupName == "" && resourceGroupID == "" {
		fmt.Print("Removing all resources not currently supported, select a resource group")
		return nil
	}
	serviceInstances, err := List()
	if err != nil {
		return err
	}
	RmServiceInstances(serviceInstances)
	return nil
}

func Tst(apikey, region string, resourceGroupName string) error {
	return TstCommon(apikey, "", "", region, resourceGroupName, "")
}

func TstCommon(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string) error {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID); err != nil {
		return err
	}
	serviceInstances, err := List()
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
	fmt.Println("tag", apikey, resourceGroup)
	return nil
}
