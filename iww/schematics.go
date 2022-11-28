package iww

import (
	"log"

	"github.com/IBM/schematics-go-sdk/schematicsv1"
)

// --- Find does does not find new resources, it does introduce a new destroy operation
type ResourceFinderSchematics struct{}

func (finder ResourceFinderSchematics) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderSchematics")
	moreInstanceWrappers = wrappedResourceInstances
	for _, ri := range wrappedResourceInstances {
		crn := ri.crn
		if crn.resourceType == "schematics" {
			if crn.vpcType == "workspace" {
				ri.operations = &SchematicsWorkspaceOpertions{}
			}
		}
	}
	return
}

func (context *Context) getSchematicsClient(crn *Crn) (client *schematicsv1.SchematicsV1, err error) {
	return schematicsv1.NewSchematicsV1(&schematicsv1.SchematicsV1Options{
		Authenticator: context.authenticator,
		URL:           ApiEndpoint("https://<region>.schematics.cloud.ibm.com", crn.region),
	})
}

//--------------------------------------
type SchematicsWorkspaceOpertions struct {
}

func (s *SchematicsWorkspaceOpertions) Fetch(si *ResourceInstanceWrapper) {
	crn := si.crn
	client, err := MustGlobalContext().getSchematicsClient(crn)
	if err != nil {
		return
	}
	if result, _, err := client.GetWorkspace(client.NewGetWorkspaceOptions(crn.vpcId)); err == nil {
		if err == nil {
			si.Name = result.Name
			si.state = SIStateExists
		} else {
			si.state = SIStateDeleted
		}
	}
}

func (s *SchematicsWorkspaceOpertions) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "schematics workspace", *si.crn)
}

func (s *SchematicsWorkspaceOpertions) Destroy(si *ResourceInstanceWrapper) {
	crn := si.crn
	client, err := MustGlobalContext().getSchematicsClient(crn)
	if err != nil {
		return
	}
	id := crn.vpcId
	_, _, err = client.DeleteWorkspace(client.NewDeleteWorkspaceOptions("", id))
	if err != nil {
		log.Print("Error deleting workspace id:", id, "name:", si.Name, "err:", err)
	}
}
