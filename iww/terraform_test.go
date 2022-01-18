package iww

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
)

func execTerraformCommand(t *testing.T, tempDir string) (string, error) {
	cmd := exec.Command("sh", "-c", "set -x; env|grep TF_VAR_; terraform version; cd "+tempDir+"; terraform init -no-color ; terraform apply -auto-approve -no-color")
	stdoutStderr_b, err := cmd.CombinedOutput()
	stdoutStderr := string(stdoutStderr_b)
	return stdoutStderr, err
}

func ListWithApikeyRegion(apikey, region string, resourceGroupName string) ([]*ResourceInstanceWrapper, error) {
	if err := SetGlobalContext(apikey, "", "", region, resourceGroupName, ""); err != nil {
		return nil, err
	}
	return List()
}
func ListWithApikey(apikey, resourceGroupName string) ([]*ResourceInstanceWrapper, error) {
	return ListWithApikeyRegion(apikey, "", resourceGroupName)
}

func testTerraform(t *testing.T, testDirectory string) {
	assert := assert.New(t)
	cwd, err := os.Getwd()
	assert.Nil(err)
	t.Log("cwd", cwd)
	tempDir, err := ioutil.TempDir("", "")
	t.Log("tempDir", tempDir)
	assert.Nil(err)
	err = copy.Copy(cwd+"/terraform_test_input/"+testDirectory, tempDir)
	assert.Nil(err)
	outerr, err := execTerraformCommand(t, tempDir)
	t.Logf("%s\n", outerr)
	assert.Nil(err)
	apikey := apikey()
	resourceGroupName := resourceGroupName()
	serviceInstances, err := ListWithApikey(apikey, resourceGroupName)
	assert.Nil(err)
	assert.NotZero(len(serviceInstances))
	err = Rm(apikey, "", resourceGroupName, "")
	assert.Nil(err)
	serviceInstances, err = ListWithApikey(apikey, resourceGroupName)
	assert.Nil(err)
	assert.Zero(len(serviceInstances))
}

func rmResourceGroup(t *testing.T, resourceGroupName string) {
	assert := assert.New(t)
	apikey := apikey()
	serviceInstances, err := ListWithApikey(apikey, resourceGroupName)
	assert.Nil(err)
	err = Rm(apikey, "", resourceGroupName, "")
	serviceInstances, err = ListWithApikey(apikey, resourceGroupName)
	assert.Len(serviceInstances, 0)
	assert.Nil(err)
}

func TestTerraformVpc(t *testing.T) {
	testTerraform(t, "vpc")
}

func TestTerraformVpcsubnet(t *testing.T) {
	testTerraform(t, "vpcsubnet")
}

func TestTerraformVpc3tier(t *testing.T) {
	testTerraform(t, "vpc3tier")
}

func TestTerraformVpcTgDns(t *testing.T) {
	testTerraform(t, "vpc-tg-dns-iam")
}

func TestTerraformRest(t *testing.T) {
	testTerraform(t, "vpc-rest")
}

func TestTerraformResources(t *testing.T) {
	testTerraform(t, "resources")
}

func apikey() string {
	return os.Getenv("TF_VAR_ibmcloud_api_key")
}
func resourceGroupName() string {
	return os.Getenv("TF_VAR_resource_group_name")
}
