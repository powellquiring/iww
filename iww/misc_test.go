package iww

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func xTestLs(t *testing.T) {
	assert := assert.New(t)
	apikey := apikey()
	err := SetGlobalContext(apikey, "", "", "", "", "", "")
	assert.Nil(err)
	context := MustGlobalContext()
	//context.crn = "crn:v1:bluemix:public:cloud-object-storage:global:a/713c783d9a507a53135fe6793c37cc74:1fd45853-1f6a-4c1c-aa43-9244d2644624::"
	serviceInstances, err := List(false)
	assert.Nil(err)
	for _, si := range serviceInstances {
		if rko, ok := si.operations.(*ResourceKeyOperations); ok {
			if *rko.getResult.SourceCRN == context.crn {
				print(rko)
			}
		}
	}
	assert.Len(serviceInstances, 1)

}

/*

func TestRmtmp(t *testing.T) {
	resourceGroupName := "tmp"
	rmResourceGroup(t, resourceGroupName)
}

func TestRmSnaps(t *testing.T) {
	resourceGroupName := "snaps"
	rmResourceGroup(t, resourceGroupName)
}

func TestRm3tier(t *testing.T) {
	resourceGroupName := "3tier"
	rmResourceGroup(t, resourceGroupName)
}

func TestRmNoCreate(t *testing.T) {
	resourceGroupName := resourceGroupName()
	rmResourceGroup(t, resourceGroupName)
}

func helpTestLs(t *testing.T, apikey, region, resourceGroupName string, fast bool) {
	assert := assert.New(t)
	err := Ls(apikey, region, resourceGroupName, "", fast)
	assert.Nil(err)
}
func TestLsFast(t *testing.T) {
	apikey := apikey()
	helpTestLs(t, apikey, "", "", true)
}


func TestLsVpcidByRG(t *testing.T) {
	apikey := apikey()
	resourceGroupName := resourceGroupName()
	assert := assert.New(t)
	err := Ls(apikey, "", resourceGroupName, "r006-2fe69b18-cfa3-46dd-8751-90dfbfef7fd9", false)
	assert.Nil(err)
}

func TestLsVpcid(t *testing.T) {
	apikey := apikey()
	assert := assert.New(t)
	err := Ls(apikey, "", "", "r006-2fe69b18-cfa3-46dd-8751-90dfbfef7fd9", false)
	assert.Nil(err)
}

func TestLsDefaultGroupUssouth(t *testing.T) {
	apikey := apikey()
	resourceGroupName := resourceGroupName()
	helpTestLs(t, apikey, "us-south", resourceGroupName, false)
}
func TestLsDefaultGroup(t *testing.T) {
	apikey := apikey()
	resourceGroupName := resourceGroupName()
	helpTestLs(t, apikey, "", resourceGroupName, false)
}

func helpTestTst(t *testing.T, apikey, region, resourceGroupName string, fast bool) {
	assert := assert.New(t)
	err := Tst(apikey, region, resourceGroupName)
	assert.Nil(err)
}
func TestTstDefaultGroup(t *testing.T) {
	apikey := apikey()
	resourceGroupName := resourceGroupName()
	helpTestTst(t, apikey, "", resourceGroupName, false)
}

var _testgetVpcService *vpcv1.VpcV1

func createVpc(assert *assert.Assertions, service *vpcv1.VpcV1, name string, resourceGroup resourcemanagerv2.ResourceGroup) *vpcv1.VPC {
	co := service.NewCreateVPCOptions()
	co.SetName(name)
	co.SetResourceGroup(&vpcv1.ResourceGroupIdentityByID{ID: resourceGroup.ID})
	result, _, err := service.CreateVPC(co)
	assert.Nil(err)
	return result
}

func getVpc(assert *assert.Assertions, service *vpcv1.VpcV1, id string) *vpcv1.VPC {
	getVpcOptions := &vpcv1.GetVPCOptions{}
	getVpcOptions.SetID(id)
	result, _, err := service.GetVPC(getVpcOptions)
	assert.Nil(err)
	return result
}

func createVpcAndWait(assert *assert.Assertions, service *vpcv1.VpcV1, name string, resourceGroup resourcemanagerv2.ResourceGroup) *vpcv1.VPC {
	vpc := getVpc(assert, service, name)
	if vpc != nil {
		return vpc
	}
	vpc = createVpc(assert, service, name, resourceGroup)
	for i := 0; i < 20; i++ {
		time.Sleep(2)
		vpc = getVpc(assert, service, *vpc.ID)
		if *vpc.Status == "available" {
			return vpc
		}
	}
	return nil
}

*/
