package iww

import "log"

// --- Find does does not find new resources, it does introduce a new destroy operation
type ResourceFinderTransitGateway struct{}

func (finder ResourceFinderTransitGateway) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderTransitGateway")
	for _, ri := range wrappedResourceInstances {
		if ri.crn.resourceType == "transit" {
			ri.operations = &TransitGatewayServiceOpertions{}
		}
	}
	moreInstanceWrappers = wrappedResourceInstances
	err = nil
	return
}

//--------------------------------------
type TransitGatewayServiceOpertions struct {
}

func (s *TransitGatewayServiceOpertions) Fetch(si *ResourceInstanceWrapper) {
	(&TypicalServiceOperations{}).Fetch(si)
}

func (s *TransitGatewayServiceOpertions) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return (&TypicalServiceOperations{}).FormatInstance(si, fast)
}

func (s *TransitGatewayServiceOpertions) Destroy(si *ResourceInstanceWrapper) {
	crn := si.crn
	if client, err := MustGlobalContext().getTransitGatewayClient(crn); err == nil {
		deleteTransitGatewayOptions := client.NewDeleteTransitGatewayOptions(
			crn.vpcId,
			// crn.Crn,
		)

		response, err := client.DeleteTransitGateway(deleteTransitGatewayOptions)
		if err != nil {
			statusCode := 0
			if response != nil {
				statusCode = response.StatusCode
				return
			}
			log.Print("TransitGateway error while deleting status code:", statusCode, ", err:", err)
		}
	} else {
		log.Print("Error deleting transit gateway err:", err)
	}
}
