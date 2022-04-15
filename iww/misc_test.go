package iww

import (
	"testing"
	"time"

	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/assert"
)

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
	err := Ls(apikey, region, resourceGroupName, fast)
	assert.Nil(err)
}
func TestLsFast(t *testing.T) {
	apikey := apikey()
	helpTestLs(t, apikey, "", "", true)
}

func TestLs(t *testing.T) {
	apikey := apikey()
	helpTestLs(t, apikey, "", "", false)
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
