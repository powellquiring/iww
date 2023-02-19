package iww

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func apikey() string {
	return os.Getenv("TF_VAR_ibmcloud_api_key")
}
func resourceGroupName() string {
	return os.Getenv("TF_VAR_resource_group_name")
}

func runCommand(dir, command string, arg ...string) (bytes.Buffer, error) {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd := exec.Command(command, arg...)
	cmd.Dir = dir
	cmd.Stdout = mw
	cmd.Stderr = mw
	err := cmd.Run()
	return stdBuffer, err
}
func execTerraformOutput(dir string) (bytes.Buffer, error) {
	return runCommand(dir, "sh", "-c", "terraform output -json")
}
func execTerraformApply(t *testing.T, tempDir string) (string, error) {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd := exec.Command("sh", "-c", "set -x; env|grep TF_VAR_; terraform version;terraform init -no-color ; terraform apply -auto-approve -no-color")
	cmd.Dir = tempDir
	cmd.Stdout = mw
	cmd.Stderr = mw

	if nottesting {
		log.Println(stdBuffer.String())
	}

	err := cmd.Run()
	return stdBuffer.String(), err
}

func listWithParams(apikey string, token string, accountID string, region string, resourceGroupName string, resourceGroupID string, vpcid string) ([]*ResourceInstanceWrapper, error) {
	if err := SetGlobalContext(apikey, token, accountID, region, resourceGroupName, resourceGroupID, vpcid, true); err != nil {
		return nil, err
	}
	fast := false
	ret, err := List(fast)
	lsOutput(ret, fast)
	return ret, err
}

func ListWithApikeyRegion(apikey, region string, resourceGroupName string) ([]*ResourceInstanceWrapper, error) {
	return listWithParams(apikey, "", "", region, resourceGroupName, "", "")
}

func ListWithApikey(apikey, resourceGroupName string) ([]*ResourceInstanceWrapper, error) {
	return ListWithApikeyRegion(apikey, "", resourceGroupName)
}

var nottesting = false

// testDirectory(dir) returns the directory, 'vpc
func testDirectory(testDirectory string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Panic("os.Getwd failed")
	}
	return cwd + "/terraform_test_input/" + testDirectory
}

// return the vpcid output string from the command: terraform output -json
func terraformOutputVpcid(buffer bytes.Buffer) string {
	type Anon_s struct {
		Value string
	}
	type Vpcid_s struct {
		Vpcid Anon_s
	}
	/*
			{
			"vpcid": {
				"sensitive": false,
				"type": "string",
				"value": "r006-5f4d2ea3-fbf6-40f5-87b2-fce648b1c872"
			}
		}
	*/
	var vpcid Vpcid_s
	json.Unmarshal(buffer.Bytes(), &vpcid)
	return vpcid.Vpcid.Value
}

func terraformCleanup(dir string) {
	tfFiles := []string{
		".terraform",
		".terraform.lock.hcl",
		"terraform.tfstate",
		"terraform.tfstate.backup",
	}
	for _, tfFile := range tfFiles {
		if err := os.RemoveAll(filepath.Join(dir, tfFile)); err != nil {
			print("failed to delete:", filepath.Join(dir, tfFile))
		}
	}
}
func testTerraformDirectory(t *testing.T, directory string) (lenServiceInstances int) {
	assert := assert.New(t)
	serviceInstances, err := listWithParams(apikey(), "", "", "", resourceGroupName(), "", "")
	assert.Len(serviceInstances, 0)
	dir := testDirectory(directory)
	defer terraformCleanup(dir)
	_, err = execTerraformApply(t, dir)
	assert.Nil(err)
	// test ls and rm using vpcid.  the vpc terraform specifies just a vpc which creates a default acl and sg
	serviceInstances, err = listWithParams(apikey(), "", "", "", resourceGroupName(), "", "")

	lenServiceInstances = len(serviceInstances)
	assert.Greater(lenServiceInstances, 0)
	Rm(apikey(), "", resourceGroupName(), "", "", "", true, true)
	serviceInstances, err = listWithParams(apikey(), "", "", "", resourceGroupName(), "", "")
	assert.Len(serviceInstances, 0)
	return lenServiceInstances
}

/*
----------------
// Test the environment variables that contain spikey and resource group name
// These tests work 9/2/2022
----------------
*/
func TestLs(t *testing.T) {
	Ls(apikey(), "", "", "", false, true)
}

func TestListWithDefaultApikeyGroupName(t *testing.T) {
	assert := assert.New(t)
	_, err := ListWithApikey(apikey(), resourceGroupName())
	assert.NoError(err)
}

/*
func TestRmWithDefaultApikeyGroupName(t *testing.T) {
	Rm(apikey(), "", resourceGroupName(), "", "", "crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image", false, true)
}
*/

func TestRmWithDefaultApikeyCrn(t *testing.T) {
	Rm(apikey(), "", "", "", "", "vpc crn:v1:bluemix:public:is:us-south:a/713c783d9a507a53135fe6793c37cc74::image:r006-1c19e164-b3b1-473f-aaed-bafa0d344ddb", false, true)
}

func TestListWithDefaultApikey(t *testing.T) {
	assert := assert.New(t)
	_, err := ListWithApikey(apikey(), "")
	assert.NoError(err)
}

func TestTerraformVpcTgDns(t *testing.T) {
	testTerraformDirectory(t, "vpc-tg-dns-iam")
}

func TestTerraformVpc(t *testing.T) {
	assert := assert.New(t)
	lenServiceInstances := testTerraformDirectory(t, "vpc")
	assert.Equal(lenServiceInstances, 3)
}

func TestTerraformVpcsubnet(t *testing.T) {
	testTerraformDirectory(t, "vpcsubnet")
}

func TestTerraformVpc3tier(t *testing.T) {
	testTerraformDirectory(t, "vpc3tier")
}

func TestTerraformRest(t *testing.T) {
	testTerraformDirectory(t, "vpc-rest")
}

func TestTerraformResources(t *testing.T) {
	testTerraformDirectory(t, "resources")
}

/*----------------

func TestTerraformVpcVpcid(t *testing.T) {
	assert := assert.New(t)
	dir := testDirectory("vpc")
	defer terraformCleanup(dir)
	_, err := execTerraformApply(t, dir)
	assert.Nil(err)
	buffer, err := execTerraformOutput(dir)
	assert.Nil(err)
	vpcid := terraformOutputVpcid(buffer)
	// test ls and rm using vpcid.  the vpc test specifies just a vpc which creates a default acl and sg
	serviceInstances, err := listWithParams(apikey(), "", "", "", "", "", vpcid)
	assert.Len(serviceInstances, 3)
	Rm(apikey(), "", "", "", vpcid)
	serviceInstances, err = listWithParams(apikey(), "", "", "", "", "", vpcid)
	assert.Len(serviceInstances, 0)
}


--------------------------------*/
