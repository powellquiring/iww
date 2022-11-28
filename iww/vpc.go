package iww

// vpc infrastructure service

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/Workiva/go-datastructures/set"
)

// --- vpc, is
type ResourceFinderVpc struct{}

func (finder ResourceFinderVpc) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderVpc")
	resourceInstances, err := readVpcExtraInstances(wrappedResourceInstances)
	MustGlobalContext().verboseLogger.Println("find ResourceFinderVpc 2")
	if err != nil {
		return nil, err
	}
	moreInstanceWrappers = append(wrappedResourceInstances, resourceInstances...)

	// replace with vpc operations for the is instances in the resource controller as well as the vpc extras
	for _, ri := range moreInstanceWrappers {
		crn := ri.crn
		if crn.resourceType == "is" {
			ri.operations, err = NewVpcOperations(crn)
			if err != nil {
				return nil, err
			}
		}
	}
	err = nil
	return
}

// These are irregular operations, notice the switch statements
type VpcSpecificVPNGatewayInstance struct{}

func (vpc VpcSpecificVPNGatewayInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteVPNGateway(service.NewDeleteVPNGatewayOptions(id))
}

func (spec VpcSpecificVPNGatewayInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	_instance, response, err := service.GetVPNGateway(service.NewGetVPNGatewayOptions(id))
	// if instance, ok := _instance.(*vpcv1.VPNGateway)
	if err == nil {
		var name string
		switch instance := _instance.(type) {
		case *vpcv1.VPNGateway:
			name = *instance.Name
		case *vpcv1.VPNGatewayPolicyMode:
			name = *instance.Name
		case *vpcv1.VPNGatewayRouteMode:
			name = *instance.Name
		}
		return name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificInstanceTemplateInstance struct{}

func (vpc VpcSpecificInstanceTemplateInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteInstanceTemplate(service.NewDeleteInstanceTemplateOptions(id))
}

func (spec VpcSpecificInstanceTemplateInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	_instance, response, err := service.GetInstanceTemplate(service.NewGetInstanceTemplateOptions(id))
	if err == nil {
		var name string
		switch instance := _instance.(type) {
		case *vpcv1.InstanceTemplate:
			name = *instance.Name
		case *vpcv1.InstanceTemplateInstanceByVolume:
			name = *instance.Name
		case *vpcv1.InstanceTemplateInstanceByImage:
			name = *instance.Name
		}
		return name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificIkePolicy struct{}

func (vpc VpcSpecificIkePolicy) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteIkePolicy(service.NewDeleteIkePolicyOptions(id))
}

func (spec VpcSpecificIkePolicy) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	ikePolicy, response, err := service.GetIkePolicy(service.NewGetIkePolicyOptions(id))
	// if instance, ok := _instance.(*vpcv1.IkePolicy)
	if err == nil {
		return *ikePolicy.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

var VpcSubtypeOperationsIrregularMap = map[string]VpcSubtypeOperations{
	"instance-template": VpcSpecificInstanceTemplateInstance{},
	"vpn":               VpcSpecificVPNGatewayInstance{},
	"ikepolicy":         VpcSpecificIkePolicy{},
}

// IS operations
type VpcGenericOperation struct {
	operations VpcSubtypeOperations // actual instance like a subnet, security group, acl, ...
	name       string
	vpcid      string
}

type VpcResourceInstanceOperations interface {
	ResourceInstanceOperations
	Vpcid() string
}

func (vpc *VpcGenericOperation) Fetch(ri *ResourceInstanceWrapper) {
	client, err := MustGlobalContext().getVpcClient(ri.crn)
	if err != nil {
		log.Print("VpcGenericOperation.Fetch, getVpcClient err:", err)
	}
	name, vpcid, found, _, err := vpc.operations.Get(client, ri.crn.vpcId)
	if found {
		// when found then name is set and err should be nil
		ri.state = SIStateExists
		if vpc.name != "" && vpc.name != name {
			panic("name of vpc resource instance has changed")
		}
		if vpc.vpcid != "" && vpc.vpcid != vpcid {
			panic("vpcid of vpc resource instance has changed")
		}
		vpc.name = name
		vpc.vpcid = vpcid
	} else {
		if err == nil {
			// when not found and err is nil then the resource was identified as not existing
			ri.state = SIStateDeleted
		} else {
			// when not found and error: something is wrong
			log.Print("VpcGenericOperation.Fetch, Get err:", err)
		}
		return
	}
}

func (vpc *VpcGenericOperation) Destroy(ri *ResourceInstanceWrapper) {
	client, err := MustGlobalContext().getVpcClient(ri.crn)
	if err != nil {
		log.Print("VpcGenericOperation.Destroy, getVpcClient err:", err)
	}
	_, err = vpc.operations.Destroy(client, ri.crn.vpcId)
	if err != nil {
		log.Print("VpcGenericOperation.Destroy Destroy err:", err)
	}
}

func (vpc *VpcGenericOperation) FormatInstance(ri *ResourceInstanceWrapper, fast bool) string {
	name := "--"
	if vpc.name != "" {
		name = vpc.name
	}
	return FormatInstance(name, "vpc", *ri.crn)
}

func (vpc *VpcGenericOperation) Vpcid() string {
	return vpc.vpcid
}

//--------------------------------------
// Some operations, like security group and acl, do not need to be deleted, deleting the vpc will auto delete them
// And deleting the default will complain.
type VpcGenericNoDeleteOperation struct {
	operations    VpcGenericOperation
	destoryCalled bool
}

func (noDelete *VpcGenericNoDeleteOperation) Fetch(ri *ResourceInstanceWrapper) {
	if ri.state == SIStateExists && noDelete.destoryCalled {
		// resource existed on the previous call, and destroy has been call then pretend like it is deleted
		ri.state = SIStateDeleted
		return
	}
	noDelete.operations.Fetch(ri)
}

func (noDelete *VpcGenericNoDeleteOperation) Destroy(ri *ResourceInstanceWrapper) {
	noDelete.destoryCalled = true
}

func (noDelete *VpcGenericNoDeleteOperation) FormatInstance(ri *ResourceInstanceWrapper, fast bool) string {
	return noDelete.operations.FormatInstance(ri, fast)
}

func (noDelete *VpcGenericNoDeleteOperation) Vpcid() string {
	return noDelete.operations.vpcid
}

//--------------------------------------
// instance-groups need the membership count set to zero before deleting.
type VpcGenericInstanceGroupOperation struct {
	operations VpcGenericOperation
}

func (vpc *VpcGenericInstanceGroupOperation) Vpcid() string {
	return vpc.operations.vpcid
}

func (vpc *VpcGenericInstanceGroupOperation) Fetch(ri *ResourceInstanceWrapper) {
	vpc.operations.Fetch(ri)
}

func instanceGroupMembershipCount(client *vpcv1.VpcV1, ri *ResourceInstanceWrapper) {
	var zero int64 = 0
	instanceGroupPatch, err := (&vpcv1.InstanceGroupPatch{
		MembershipCount: &zero,
	}).AsPatch()
	if err != nil {
		log.Print("VpcGenericInstanceGroupOperation.Destroy, AsPatch err:", err)
	} else {
		ops := client.NewUpdateInstanceGroupOptions(ri.crn.vpcId, instanceGroupPatch)
		_, _, err := client.UpdateInstanceGroup(ops)
		if err != nil {
			log.Print("VpcGenericInstanceGroupOperation.Destroy, UpdateInstanceGroup err:", err)
		}
	}
}

func instanceGroupManagerDelete(client *vpcv1.VpcV1, ri *ResourceInstanceWrapper) {
	result, _, err := client.ListInstanceGroupManagers(client.NewListInstanceGroupManagersOptions(ri.crn.vpcId))
	if err != nil {
		log.Print("VpcGenericInstanceGroupOperation.Destroy, ListInstanceGropupManagers err:", err)
		return
	}
	for _, _manager := range result.Managers {
		var managerId string
		switch manager := _manager.(type) {
		case *vpcv1.InstanceGroupManagerAutoScale:
			managerId = *manager.ID
		case *vpcv1.InstanceGroupManagerScheduled:
			managerId = *manager.ID
		case *vpcv1.InstanceGroupManager:
			managerId = *manager.ID
		}
		_, err = client.DeleteInstanceGroupManager(client.NewDeleteInstanceGroupManagerOptions(ri.crn.vpcId, managerId))
		if err != nil {
			log.Print("VpcGenericInstanceGroupOperation.Destroy, DeleteInstanceGroupManager err:", err)
		}
	}
}

func (vpc *VpcGenericInstanceGroupOperation) Destroy(ri *ResourceInstanceWrapper) {
	client, err := MustGlobalContext().getVpcClient(ri.crn)
	if err != nil {
		log.Print("VpcGenericInstanceGroupOperation.Destroy, getVpcClient err:", err)
	}
	instanceGroupMembershipCount(client, ri)
	instanceGroupManagerDelete(client, ri)
	vpc.operations.Destroy(ri)
}

func (vpc *VpcGenericInstanceGroupOperation) FormatInstance(ri *ResourceInstanceWrapper, fast bool) string {
	return vpc.operations.FormatInstance(ri, fast)
}

type VpcSpecificVPCInstanceWrapper struct {
}

func (vpc *VpcSpecificVPCInstanceWrapper) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return VpcSubtypeOperationsMap["vpc"].Destroy(service, id)
}

func (spec *VpcSpecificVPCInstanceWrapper) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	name, _, found, response, err := VpcSubtypeOperationsMap["vpc"].Get(service, id)
	return name, id, found, response, err
}

//--------------------------------------
func NewVpcOperations(crn *Crn) (ResourceInstanceOperations, error) {
	if crn.resourceType != "is" {
		return nil, errors.New("crn is not is: " + crn.vpcType)
	}
	if specificInstance, ok := VpcSubtypeOperationsMap[crn.vpcType]; ok {
		// typical
		genericOperation := &VpcGenericOperation{
			operations: specificInstance,
		}
		// a few special case wrappers around the standard operations
		switch crn.vpcType {
		case "network-acl", "security-group":
			return &VpcGenericNoDeleteOperation{
				operations: *genericOperation,
			}, nil
		case "instance-group":
			return &VpcGenericInstanceGroupOperation{
				operations: *genericOperation,
			}, nil
		case "vpc":
			genericOperation.operations = &VpcSpecificVPCInstanceWrapper{}
			return genericOperation, nil
		default:
			return genericOperation, nil
		}
	} else if specificInstance, ok := VpcSubtypeOperationsIrregularMap[crn.vpcType]; ok {
		return &VpcGenericOperation{
			operations: specificInstance,
		}, nil
	} else {
		return UnimplementedServiceOperations{}, nil
	}
}

func isSubnet(ri *ResourceInstanceWrapper) bool {
	return ri.crn.resourceType == "is" && ri.crn.vpcType == "subnet"
}

type resourceInstanceWrapperErr struct {
	ri  *ResourceInstanceWrapper
	err error
}

// listInstanceTemplates appens onto list the list of instance templates
//func listInstanceTemplates(list list.PersistentList, client *vpcv1.VpcV1, wg *sync.WaitGroup) {
func listInstanceTemplates(list *set.Set, client *vpcv1.VpcV1, wg *sync.WaitGroup) {
	defer wg.Done()
	lriOptions := client.NewListInstanceTemplatesOptions()
	result, _, err := client.ListInstanceTemplates(lriOptions)
	if err == nil {
		for _, t := range result.Templates {
			if it, ok := t.(*vpcv1.InstanceTemplate); ok {
				list.Add(&resourceInstanceWrapperErr{NewResourceInstanceWrapper(NewCrn(*it.CRN), it.ResourceGroup.ID, it.Name), nil})
			}
		}
	} else {
		list.Add(&resourceInstanceWrapperErr{nil, err})
	}
}

// listInstanceTemplates appens onto list the list of instance templates
func listIkePolicies(list *set.Set, client *vpcv1.VpcV1, wg *sync.WaitGroup) {
	defer wg.Done()
	// todo ID is not a CRN below
	likeOptions := client.NewListIkePoliciesOptions()
	ikePolilcies, _, err := client.ListIkePolicies(likeOptions)
	if err == nil {
		for _, it := range ikePolilcies.IkePolicies {
			region := regionFromUrl(client.Service.Options.URL)
			crn := NewFakeCrn("is", "", "ikepolicy", *it.ID, region)
			list.Add(&resourceInstanceWrapperErr{NewResourceInstanceWrapper(crn, it.ResourceGroup.ID, it.Name), nil})
		}
	} else {
		list.Add(&resourceInstanceWrapperErr{nil, err})
	}
}

var check bool = false

func regionNames(context *Context) (map[string]string, error) {
	regions := map[string]string{
		"au-syd":   "au-syd",
		"br-sao":   "br-sao",
		"ca-tor":   "ca-tor",
		"eu-de":    "eu-de",
		"eu-gb":    "eu-gb",
		"jp-osa":   "jp-osa",
		"jp-tok":   "jp-tok",
		"us-east":  "us-east",
		"us-south": "us-south",
	}
	if check {
		client, err := context.getVpcClientFromRegion("us-south")
		if err != nil {
			return nil, err
		}
		regionCollection, _, err := client.ListRegions(client.NewListRegionsOptions())
		if err != nil {
			return nil, err
		}
		if len(regionCollection.Regions) != len(regions) {
			return nil, errors.New("vpc region check mismatch: len")
		}
		for _, region := range regionCollection.Regions {
			if _, ok := regions[*region.Name]; !ok {
				return nil, errors.New("vpc region check mismatch: not found:" + *region.Name)
			}
		}
	}
	return regions, nil
}

func vpcRegionClients() ([]*vpcv1.VpcV1, error) {
	context := MustGlobalContext()
	regions, err := regionNames(context)
	if err != nil {
		return nil, err
	}
	regionClients := make([]*vpcv1.VpcV1, 0)
	for _, region := range regions {
		client, err := context.getVpcClientFromRegion(region)
		/* write a bug report region.Endpoint is https://au-syd.iaas.cloud.ibm.com expecting https://au-syd.iaas.cloud.ibm.com/v1
		client, err = vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: MustGlobalContext().authenticator,
			URL:           *region.Endpoint,
		})
		*/
		if err != nil {
			return nil, err
		}
		regionClients = append(regionClients, client)
	}
	return regionClients, nil
}

func readVpcExtraInstances(currentResourceInstances []*ResourceInstanceWrapper) ([]*ResourceInstanceWrapper, error) {
	regionClients, err := vpcRegionClients()
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	set := set.New()
	var wg sync.WaitGroup

	for _, client := range regionClients {
		time.Sleep(10 * time.Millisecond) // avoid rate limiting
		wg.Add(2)
		if Async {
			// go listInstanceTemplates(list, client, &wg)
			go listInstanceTemplates(set, client, &wg)
			go listIkePolicies(set, client, &wg)
		} else {
			// listInstanceTemplates(list, client, &wg)
			listInstanceTemplates(set, client, &wg)
			listIkePolicies(set, client, &wg)
		}

	}
	wg.Wait()

	for _, _rie := range set.Flatten() {
		rie := _rie.(*resourceInstanceWrapperErr)
		if rie.err != nil {
			return nil, err
		}
		wrappedResourceInstances = append(wrappedResourceInstances, rie.ri)
	}
	return wrappedResourceInstances, nil
}

// regionFromUrl returns the region in a url string: "https://us-south.iaas.cloud.ibm.com/v1"
func regionFromUrl(url string) string {
	rest := strings.Split(url, "//")
	regionPlus := strings.Split(rest[1], ".")
	return regionPlus[0]
}
