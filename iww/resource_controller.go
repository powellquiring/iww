package iww

import (
	"fmt"
	"log"
	"strconv"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
)

// ------------------------------------
// Global variable initialization section
func (context *Context) getResourceControllerClient() (client *resourcecontrollerv2.ResourceControllerV2, err error) {
	return resourcecontrollerv2.NewResourceControllerV2(&resourcecontrollerv2.ResourceControllerV2Options{
		Authenticator: context.authenticator,
	})
}

// --- Resource controller is the set of cloud tracked resources.  Almost all of these are in the resources view in the cloud console
type ResourceFinderRC struct{}

func (finder ResourceFinderRC) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	context := MustGlobalContext()
	resourceControllerClient, err := context.getResourceControllerClient()
	if err != nil {
		return nil, err
	}
	context.verboseLogger.Println("find ResourceFinderRC")
	resourceInstances, err := readResourceInstances(resourceControllerClient)
	if err != nil {
		return nil, err
	}
	for _, ri := range resourceInstances {
		ri.operations = &TypicalServiceOperations{}
	}
	moreInstanceWrappers = append(wrappedResourceInstances, resourceInstances...)
	err = nil
	return
}

// --- Resource keys
type ResourceFinderResourceKeys struct{}

func (finder ResourceFinderResourceKeys) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	context := MustGlobalContext()
	context.verboseLogger.Println("find ResourceFinderResourceKeys")
	resourceControllerClient, err := context.getResourceControllerClient()
	resourceInstances, err := readResourceKeys(resourceControllerClient, wrappedResourceInstances)
	if err != nil {
		return nil, err
	}
	for _, ri := range resourceInstances {
		ri.operations = &ResourceKeyOperations{}
	}
	moreInstanceWrappers = append(wrappedResourceInstances, resourceInstances...)
	err = nil
	return
}

// --- resources section instances

// Read the resource instances from the resource controller
func readResourceInstances(resourceControllerClient *resourcecontrollerv2.ResourceControllerV2) ([]*ResourceInstanceWrapper, error) {
	context := MustGlobalContext()
	var resourceInstances []resourcecontrollerv2.ResourceInstance
	var err error
	if context.crn != "" {
		resourceInstances, err = readResourceInstance(context.crn)
	} else {
		// filter by resource group
		lriOptions := resourceControllerClient.NewListResourceInstancesOptions()
		if context.resourceGroupID != "" {
			lriOptions.SetResourceGroupID(context.resourceGroupID)
		}
		resourceInstances, err = ResourceInstances(resourceControllerClient, lriOptions)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
	}

	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	for _, ri := range resourceInstances {
		crn := NewCrn(*ri.CRN)
		si := NewResourceInstanceWrapper(crn, ri.ResourceGroupID, ri.Name)
		// filter by region
		if context.region == "" || context.region == crn.region {
			wrappedResourceInstances = append(wrappedResourceInstances, si)
		}
	}
	return wrappedResourceInstances, nil
}

// Return the list of resource instances matching the option
func ResourceInstances(service *resourcecontrollerv2.ResourceControllerV2, lrio *resourcecontrollerv2.ListResourceInstancesOptions) ([]resourcecontrollerv2.ResourceInstance, error) {
	resourceInstances := make([]resourcecontrollerv2.ResourceInstance, 0)
	// limit the number of calls
	for i := 0; i < 100; i++ {
		resourceInstancesList, _, err := service.ListResourceInstances(lrio)
		if err != nil {
			return resourceInstances, err
		}
		resourceInstances = append(resourceInstances, resourceInstancesList.Resources...)
		if resourceInstancesList.NextURL == nil {
			break // yeah, got them all
		}
		startString, err := core.GetQueryParam(resourceInstancesList.NextURL, "start")
		if err != nil {
			return resourceInstances, err
		}
		lrio.SetStart(*startString)
	}
	return resourceInstances, nil
}

// readResourceInstance returns a slice containing one resource matching the provided crn
func readResourceInstance(crn string) ([]resourcecontrollerv2.ResourceInstance, error) {
	context := MustGlobalContext()
	c := NewCrn(crn)
	id := c.id

	getResourceInstanceOptions := context.resourceControllerClient.NewGetResourceInstanceOptions(id)
	resourceInstance, _, err := context.resourceControllerClient.GetResourceInstance(getResourceInstanceOptions)
	if err != nil {
		return nil, err
	}
	return []resourcecontrollerv2.ResourceInstance{*resourceInstance}, nil
}

//-- resourcd keys section

// Return the list of service keys matching the option
func ResourceKeys(service *resourcecontrollerv2.ResourceControllerV2, lrio *resourcecontrollerv2.ListResourceKeysOptions) ([]resourcecontrollerv2.ResourceKey, error) {
	resourceKeys := make([]resourcecontrollerv2.ResourceKey, 0)
	// limit the number of calls
	for i := 0; i < 100; i++ {
		resourceInstancesList, _, err := service.ListResourceKeys(lrio)
		if err != nil {
			return resourceKeys, err
		}
		resourceKeys = append(resourceKeys, resourceInstancesList.Resources...)
		if resourceInstancesList.NextURL == nil {
			break // yeah, got them all
		}
		startString, err := core.GetQueryParam(resourceInstancesList.NextURL, "start")
		if err != nil {
			return resourceKeys, err
		}
		lrio.SetStart(*startString)
	}
	return resourceKeys, nil
}

// readResourceKeys will return wrapped keys for the resources in the list
func readResourceKeys(resourceControllerClient *resourcecontrollerv2.ResourceControllerV2, justTheseResources []*ResourceInstanceWrapper) ([]*ResourceInstanceWrapper, error) {
	context := MustGlobalContext()
	lrkOptions := resourceControllerClient.NewListResourceKeysOptions()
	if context.resourceGroupID != "" {
		lrkOptions.SetResourceGroupID(context.resourceGroupID)
	}
	resourceKeys, err := ResourceKeys(resourceControllerClient, lrkOptions)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	justTheseResourcesCrns := make(map[string]bool)
	for _, justThisOne := range justTheseResources {
		justTheseResourcesCrns[(*justThisOne.crn).Crn] = true
	}

	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	var lastErr error
	for _, rk := range resourceKeys {
		crn_s := *rk.CRN
		if justTheseResourcesCrns[*rk.SourceCRN] {
			crn := NewCrn(crn_s)
			si := NewResourceInstanceWrapper(crn, rk.ResourceGroupID, rk.Name)
			if err != nil {
				lastErr = err
				fmt.Println("BAD CRN:", crn_s)
			} else {
				if context.region == "" || context.region == crn.region {
					wrappedResourceInstances = append(wrappedResourceInstances, si)
				}
			}
		}
	}
	return wrappedResourceInstances, lastErr
}

// --------------------------------------
type ResourceKeyOperations struct {
	getResult   *resourcecontrollerv2.ResourceKey
	getResponse *core.DetailedResponse
	getErr      error
}

func (s *ResourceKeyOperations) Destroy(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewDeleteResourceKeyOptions(id)
	response, err := rc.DeleteResourceKey(options)
	if err != nil {
		statusCode := "not_returned"
		if response != nil {
			statusCode = strconv.Itoa(response.StatusCode)
		}
		log.Print("TypicalServiceOpertions DeleteResourceKey, StatusCode:", statusCode, " Crn:", si.crn.Crn, " err:", err.Error())
	}
}

func (s *ResourceKeyOperations) Fetch(si *ResourceInstanceWrapper) {
	context := MustGlobalContext()
	id := si.crn.Crn
	rc := context.resourceControllerClient
	options := rc.NewGetResourceKeyOptions(id)
	s.getResult, s.getResponse, s.getErr = rc.GetResourceKey(options)
	if s.getErr != nil {
		if s.getResponse != nil && (s.getResponse.StatusCode == 404 || s.getResponse.StatusCode == 410) {
			si.state = SIStateDeleted
		} else {
			log.Print(s.getErr)
		}
	} else {
		si.state = SIStateExists
		if s.getResult != nil && *s.getResult.State == "removed" {
			si.state = SIStateDeleted
		}
	}
}

func (s *ResourceKeyOperations) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "-", *si.crn)
}
