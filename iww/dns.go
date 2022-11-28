package iww

// todo custom resolver locations

import (
	"log"

	"github.com/IBM/networking-go-sdk/dnssvcsv1"
)

// Finder
type ResourceFinderDns struct{}

// Find the DNS resources that are not in the RC
func (finder ResourceFinderDns) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderDns")
	resourceInstances, err := readDnsResources(wrappedResourceInstances)
	if err != nil {
		return nil, err
	}
	moreInstanceWrappers = append(wrappedResourceInstances, resourceInstances...)
	err = nil
	return
}

func isDns(ri *ResourceInstanceWrapper) bool {
	return ri.crn.resourceType == "dns-svcs"
}

func (context *Context) getDnssvcsClient() (client *dnssvcsv1.DnsSvcsV1, err error) {
	return dnssvcsv1.NewDnsSvcsV1(&dnssvcsv1.DnsSvcsV1Options{
		Authenticator: context.authenticator,
	})
}

// Read the zones, todo rest of the dns stypes like custom locations
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

			pools, _, err := client.ListPools(client.NewListPoolsOptions(ri.crn.id))
			if err != nil {
				return nil, err
			}
			for _, pool := range pools.Pools {
				wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "pool", *pool.ID, pool.Name, &DnsPool{}))
			}

			monitors, _, err := client.ListMonitors(client.NewListMonitorsOptions(ri.crn.id))
			if err != nil {
				return nil, err
			}
			for _, monitor := range monitors.Monitors {
				wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "monitor", *monitor.ID, monitor.Name, &DnsMonitor{}))
			}

			customResolvers, _, err := client.ListCustomResolvers(client.NewListCustomResolversOptions(ri.crn.id))
			if err != nil {
				return nil, err
			}
			for _, customResolver := range customResolvers.CustomResolvers {
				wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "cr", *customResolver.ID, customResolver.Name, &DnsCustomResolver{}))
			}

			for _, sub := range result.Dnszones {
				wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "zone", *sub.ID, sub.Name, &Dnszone{}))
				pns, _, err := client.ListPermittedNetworks(client.NewListPermittedNetworksOptions(ri.crn.id, *sub.ID))
				if err != nil {
					return nil, err
				}
				for _, pn := range pns.PermittedNetworks {
					// todo notice the name is actually the ID of the dns zone which is needed to delete, kludge city
					wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "pn", *pn.ID, sub.ID, &DnsPermittedNetwork{}))
				}
				lbs, _, err := client.ListLoadBalancers(client.NewListLoadBalancersOptions(ri.crn.id, *sub.ID))
				if err != nil {
					return nil, err
				}
				for _, lb := range lbs.LoadBalancers {
					wrappedResourceInstances = append(wrappedResourceInstances, NewSubInstance(ri, "lb", *lb.ID, sub.ID, &DnsLoadBalancer{}))
				}
			}
		}
	}
	return wrappedResourceInstances, nil
}

// Zone operations
type Dnszone struct {
}

func (dzone *Dnszone) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	result, _, err := client.GetDnszone(client.NewGetDnszoneOptions(si.crn.id, si.crn.vpcId))
	if err == nil {
		si.Name = result.Name
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (dzone *Dnszone) Destroy(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	_, err = client.DeleteDnszone(client.NewDeleteDnszoneOptions(si.crn.id, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (dzone *Dnszone) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "dns", *si.crn)
}

type DnsPool struct {
}

func (pool *DnsPool) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	result, _, err := client.GetPool(client.NewGetPoolOptions(si.crn.id, si.crn.vpcId))
	if err == nil {
		si.Name = result.Name
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (pool *DnsPool) Destroy(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	_, err = client.DeletePool(client.NewDeletePoolOptions(si.crn.id, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (pool *DnsPool) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "dns", *si.crn)
}

type DnsMonitor struct {
}

func (pool *DnsMonitor) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	result, _, err := client.GetMonitor(client.NewGetMonitorOptions(si.crn.id, si.crn.vpcId))
	if err == nil {
		si.Name = result.Name
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (pool *DnsMonitor) Destroy(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	_, err = client.DeleteMonitor(client.NewDeleteMonitorOptions(si.crn.id, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (pool *DnsMonitor) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "dns", *si.crn)
}

type DnsCustomResolver struct {
}

func (customResolver *DnsCustomResolver) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	result, _, err := client.GetCustomResolver(client.NewGetCustomResolverOptions(si.crn.id, si.crn.vpcId))
	if err == nil {
		si.Name = result.Name
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (customResolver *DnsCustomResolver) Destroy(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	// disable custom resolver not normal stuff /////
	cro := client.NewUpdateCustomResolverOptions(si.crn.id, si.crn.vpcId)
	cro.SetEnabled(false)
	_, _, err = client.UpdateCustomResolver(cro)
	if err != nil {
		log.Print(err)
	}
	// normal
	_, err = client.DeleteCustomResolver(client.NewDeleteCustomResolverOptions(si.crn.id, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (customResolver *DnsCustomResolver) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "dns", *si.crn)
}

// 3 parameter //////////////////////////////////////////////////////////////////////

// Permitted network operations
type DnsPermittedNetwork struct {
}

func (pn *DnsPermittedNetwork) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	// the si.Name is actually the zone id
	_, _, err = client.GetPermittedNetwork(client.NewGetPermittedNetworkOptions(si.crn.id, *si.Name, si.crn.vpcId))
	if err == nil {
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (pn *DnsPermittedNetwork) Destroy(si *ResourceInstanceWrapper) {
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	_, _, err = client.DeletePermittedNetwork(client.NewDeletePermittedNetworkOptions(si.crn.id, *si.Name, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (pn *DnsPermittedNetwork) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance("zoneid-"+*si.Name, "dns", *si.crn)
}

// Permitted network operations
type DnsLoadBalancer struct {
}

func (lb *DnsLoadBalancer) Fetch(si *ResourceInstanceWrapper) { // fetch from cloud and upate the state, no need to retry in Fetch
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	// the si.Name is actually the zone id
	_, _, err = client.GetLoadBalancer(client.NewGetLoadBalancerOptions(si.crn.id, *si.Name, si.crn.vpcId))
	if err == nil {
		si.state = SIStateExists
	} else {
		si.state = SIStateDeleted
	}
}
func (lb *DnsLoadBalancer) Destroy(si *ResourceInstanceWrapper) {
	client, err := MustGlobalContext().getDnssvcsClient()
	if err != nil {
		log.Print(err)
		return
	}
	_, err = client.DeleteLoadBalancer(client.NewDeleteLoadBalancerOptions(si.crn.id, *si.Name, si.crn.vpcId))
	if err != nil {
		log.Print(err)
	}
}
func (lb *DnsLoadBalancer) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance("zoneid-"+*si.Name, "dns", *si.crn)
}
