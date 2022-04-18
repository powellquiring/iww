package iww

// vpc infrastructure service

import (
	"errors"
	"log"

	"github.com/IBM/vpc-go-sdk/vpcv1"
)

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

var VpcSubtypeOperationsIrregularMap = map[string]VpcSubtypeOperations{
	"instance-template": VpcSpecificInstanceTemplateInstance{},
	"vpn":               VpcSpecificVPNGatewayInstance{},
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

func readVpcExtraInstances(currentResourceInstances []*ResourceInstanceWrapper) ([]*ResourceInstanceWrapper, error) {
	// find all regions that contain subnets
	subnetClients := make(map[*vpcv1.VpcV1]*ResourceInstanceWrapper, 0)
	for _, ri := range currentResourceInstances {
		if isSubnet(ri) {
			client, err := MustGlobalContext().getVpcClient(ri.crn)
			if err != nil {
				return nil, err
			}
			if _, ok := subnetClients[client]; !ok {
				subnetClients[client] = ri
			}
		}
	}

	// Kludge: only add the resources in subnets already in the list of resources
	// TODO fix this, or wait for https://bigblue.aha.io/ideas/ideas/CPS-I-1597
	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	for subnetClient := range subnetClients {
		lriOptions := subnetClient.NewListInstanceTemplatesOptions()
		result, _, err := subnetClient.ListInstanceTemplates(lriOptions)
		if err != nil {
			return nil, err
		}
		for _, t := range result.Templates {
			if it, ok := t.(*vpcv1.InstanceTemplate); ok {
				wrappedResourceInstances = append(wrappedResourceInstances, NewResourceInstanceWrapper(NewCrn(*it.CRN), it.ResourceGroup.ID, it.Name))
			}
		}
	}
	return wrappedResourceInstances, nil
}
