package iww

import "github.com/IBM/networking-go-sdk/dnssvcsv1"

func isDns(ri *ResourceInstanceWrapper) bool {
	return ri.crn.resourceType == "dns-svcs"
}

func readDnsResources(currentResourceInstances []*ResourceInstanceWrapper) ([]*ResourceInstanceWrapper, error) {
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		return nil, err
	}
	wrappedResourceInstances := make([]*ResourceInstanceWrapper, 0)
	for _, ri := range currentResourceInstances {
		if isDns(ri) {
			ldos := client.NewListDnszonesOptions(ri.crn.id)
			result, _, err := client.ListDnszones(ldos)
			if err != nil {
				return nil, err
			}

			for _, dz := range result.Dnszones {
				dzCrn := *ri.crn
				dzCrn.vpcType = "dns-zone"
				dzCrn.vpcId = *dz.ID
				wrappedResourceInstances = append(wrappedResourceInstances, NewResourceInstanceWrapper(&dzCrn, ri.ResourceGroupID, dz.Name))
			}
		}
	}
	return wrappedResourceInstances, nil
}

type DnsSpecificDnszone struct {
}

func (vpc *DnsSpecificDnszone) Destroy(service *dnssvcsv1.DnsSvcsV1, id string) (interface{}, error) {
	return service.DeleteDnszone(service.NewDeleteDnszoneOptions(id, id))
}

func (spec *DnsSpecificDnszone) Get(service *dnssvcsv1.DnsSvcsV1, id string) (string, string, bool, interface{}, error) {
	instance, response, err := service.GetDnszone(service.NewGetDnszoneOptions(id, id))
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
