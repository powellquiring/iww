package main

import (
	"os"
	"text/template"
)

type subtypeToBasename struct {
	Subtype  string
	Basename string
}

var VpcSubtypeOperationsMap = []subtypeToBasename{
	{"vpc", "VPC"},
	{"subnet", "Subnet"},
	{"instance", "Instance"},
	{"volume", "Volume"},
	{"key", "Key"},
	{"load-balancer", "LoadBalancer"},
	{"floating-ip", "FloatingIP"},
	{"image", "Image"},
	{"public-gateway", "PublicGateway"},
	{"network-acl", "NetworkACL"},
	{"security-group", "SecurityGroup"},
	{"flow-log-collector", "FlowLogCollector"},
	{"instance-group", "InstanceGroup"},
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
	Get(service *vpcv1.VpcV1, id string) ( /*name*/ string /*found*/, bool /*response*/, interface{}, error)
	Destroy(service *vpcv1.VpcV1, id string) ( /*response*/ interface{}, error)
}


{{range . }}
type VpcSpecific{{ .Basename }}Instance struct{}

func (vpc VpcSpecific{{ .Basename }}Instance) Destroy(service *vpcv1.VpcV1, id string) (interface{}, error) {
	return service.Delete{{ .Basename }}(service.NewDelete{{ .Basename }}Options(id))
}

func (spec VpcSpecific{{ .Basename }}Instance) Get(service *vpcv1.VpcV1, id string) (string, bool, interface{}, error) {
	instance, response, err := service.Get{{ .Basename }}(service.NewGet{{ .Basename }}Options(id))
	if err == nil {
		return *instance.Name, true, response, nil
	} else {
		if response != nil && response.StatusCode == 404 {
			return "", false, response, nil
		} else {
			return "", false, response, err
		}
	}
}
{{ end }}
var VpcSubtypeOperationsMap = map[string]VpcSubtypeOperations{
{{- range .}}
	"{{ .Subtype }}" : VpcSpecific{{ .Basename }}Instance{},
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
