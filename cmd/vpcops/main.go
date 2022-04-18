package main

import (
	"os"
	"text/template"
)

type subtypeToBasename struct {
	Subtype  string
	Basename string
	InVpc    bool
}

var VpcSubtypeOperationsMap = []subtypeToBasename{
	{"vpc", "VPC", false},
	{"subnet", "Subnet", true},
	{"instance", "Instance", true},
	{"volume", "Volume", false},
	{"key", "Key", false},
	{"load-balancer", "LoadBalancer", false},
	{"floating-ip", "FloatingIP", false},
	{"image", "Image", false},
	{"public-gateway", "PublicGateway", true},
	{"network-acl", "NetworkACL", true},
	{"security-group", "SecurityGroup", true},
	{"flow-log-collector", "FlowLogCollector", true},
	{"instance-group", "InstanceGroup", true},
	{"snapshot", "Snapshot", false},
	// {"vpn", "VPNGateway"},
	// {"instance-template", "InstanceTemplate"},
}

var operation = `
package iww

// Generated file do not edit.  See cmd/vpcops: Makefile and main.go

/*
vpc operations for each vpc type of resource, vpc, subnet, instance, ...
*/

import (
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

type VpcSubtypeOperations interface {
	Get(service *vpcv1.VpcV1, id string) ( /*name*/ string, /*vpcid*/ string, /*found*/ bool, /*response*/ interface{}, error)
	Destroy(service *vpcv1.VpcV1, id string) ( /*response*/ interface{}, error)
}

{{range . }}
type VpcSpecific{{ .Basename }}Instance struct{}

func (vpc *VpcSpecific{{ .Basename }}Instance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.Delete{{ .Basename }}(service.NewDelete{{ .Basename }}Options(id))
}

func (spec *VpcSpecific{{ .Basename }}Instance) Get(service *vpcv1.VpcV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.Get{{ .Basename }}(service.NewGet{{ .Basename }}Options(id))
	if err == nil {
		return *instance.Name, {{if (.InVpc)}} *instance.VPC.ID {{else}} "" {{end}}, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", "", false, response, nil
		} else {
			return "", "", false, response, err
		}
	}
}
{{ end }}
var VpcSubtypeOperationsMap = map[string]VpcSubtypeOperations{
{{- range .}}
	"{{ .Subtype }}" : &VpcSpecific{{ .Basename }}Instance{},
{{- end }}
}
`

func main() {
	tmpl, err := template.New("test").Parse(operation)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, VpcSubtypeOperationsMap)
	if err != nil {
		panic(err)
	}
}
