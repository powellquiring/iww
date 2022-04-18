package iww

// Generated file do not edit.  See cmd/vpcops: Makefile and main.go

/*
vpc operations for each vpc type of resource, vpc, subnet, instance, ...
*/

import (
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

type VpcSubtypeOperations interface {
	Get(service *vpcv1.VpcV1, id string) ( /*name*/ string /*vpcid*/, string /*found*/, bool /*response*/, interface{}, error)
	Destroy(service *vpcv1.VpcV1, id string) ( /*response*/ interface{}, error)
}

type VpcSpecificVPCInstance struct{}

func (vpc *VpcSpecificVPCInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteVPC(service.NewDeleteVPCOptions(id))
}

func (spec *VpcSpecificVPCInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetVPC(service.NewGetVPCOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificSubnetInstance struct{}

func (vpc *VpcSpecificSubnetInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteSubnet(service.NewDeleteSubnetOptions(id))
}

func (spec *VpcSpecificSubnetInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetSubnet(service.NewGetSubnetOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificInstanceInstance struct{}

func (vpc *VpcSpecificInstanceInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteInstance(service.NewDeleteInstanceOptions(id))
}

func (spec *VpcSpecificInstanceInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetInstance(service.NewGetInstanceOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificVolumeInstance struct{}

func (vpc *VpcSpecificVolumeInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteVolume(service.NewDeleteVolumeOptions(id))
}

func (spec *VpcSpecificVolumeInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetVolume(service.NewGetVolumeOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificKeyInstance struct{}

func (vpc *VpcSpecificKeyInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteKey(service.NewDeleteKeyOptions(id))
}

func (spec *VpcSpecificKeyInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetKey(service.NewGetKeyOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificLoadBalancerInstance struct{}

func (vpc *VpcSpecificLoadBalancerInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteLoadBalancer(service.NewDeleteLoadBalancerOptions(id))
}

func (spec *VpcSpecificLoadBalancerInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetLoadBalancer(service.NewGetLoadBalancerOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificFloatingIPInstance struct{}

func (vpc *VpcSpecificFloatingIPInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteFloatingIP(service.NewDeleteFloatingIPOptions(id))
}

func (spec *VpcSpecificFloatingIPInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetFloatingIP(service.NewGetFloatingIPOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificImageInstance struct{}

func (vpc *VpcSpecificImageInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteImage(service.NewDeleteImageOptions(id))
}

func (spec *VpcSpecificImageInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetImage(service.NewGetImageOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificPublicGatewayInstance struct{}

func (vpc *VpcSpecificPublicGatewayInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeletePublicGateway(service.NewDeletePublicGatewayOptions(id))
}

func (spec *VpcSpecificPublicGatewayInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetPublicGateway(service.NewGetPublicGatewayOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificNetworkACLInstance struct{}

func (vpc *VpcSpecificNetworkACLInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteNetworkACL(service.NewDeleteNetworkACLOptions(id))
}

func (spec *VpcSpecificNetworkACLInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetNetworkACL(service.NewGetNetworkACLOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificSecurityGroupInstance struct{}

func (vpc *VpcSpecificSecurityGroupInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteSecurityGroup(service.NewDeleteSecurityGroupOptions(id))
}

func (spec *VpcSpecificSecurityGroupInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetSecurityGroup(service.NewGetSecurityGroupOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificFlowLogCollectorInstance struct{}

func (vpc *VpcSpecificFlowLogCollectorInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteFlowLogCollector(service.NewDeleteFlowLogCollectorOptions(id))
}

func (spec *VpcSpecificFlowLogCollectorInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetFlowLogCollector(service.NewGetFlowLogCollectorOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificInstanceGroupInstance struct{}

func (vpc *VpcSpecificInstanceGroupInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteInstanceGroup(service.NewDeleteInstanceGroupOptions(id))
}

func (spec *VpcSpecificInstanceGroupInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetInstanceGroup(service.NewGetInstanceGroupOptions(id))
	if err == nil {
		return *instance.Name, *instance.VPC.ID, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

type VpcSpecificSnapshotInstance struct{}

func (vpc *VpcSpecificSnapshotInstance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.DeleteSnapshot(service.NewDeleteSnapshotOptions(id))
}

func (spec *VpcSpecificSnapshotInstance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetSnapshot(service.NewGetSnapshotOptions(id))
	if err == nil {
		return *instance.Name, "", true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}

var VpcSubtypeOperationsMap = map[string]VpcSubtypeOperations{
	"vpc":                &VpcSpecificVPCInstance{},
	"subnet":             &VpcSpecificSubnetInstance{},
	"instance":           &VpcSpecificInstanceInstance{},
	"volume":             &VpcSpecificVolumeInstance{},
	"key":                &VpcSpecificKeyInstance{},
	"load-balancer":      &VpcSpecificLoadBalancerInstance{},
	"floating-ip":        &VpcSpecificFloatingIPInstance{},
	"image":              &VpcSpecificImageInstance{},
	"public-gateway":     &VpcSpecificPublicGatewayInstance{},
	"network-acl":        &VpcSpecificNetworkACLInstance{},
	"security-group":     &VpcSpecificSecurityGroupInstance{},
	"flow-log-collector": &VpcSpecificFlowLogCollectorInstance{},
	"instance-group":     &VpcSpecificInstanceGroupInstance{},
	"snapshot":           &VpcSpecificSnapshotInstance{},
}
