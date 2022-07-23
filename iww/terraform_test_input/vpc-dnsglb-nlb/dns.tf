resource "ibm_resource_instance" "dns" {
  name              = "${var.prefix}-dns"
  resource_group_id = data.ibm_resource_group.all_rg.id
  location          = "global"
  service           = "dns-svcs"
  plan              = "standard-dns"
}

resource "ibm_dns_zone" "widgets_cogs" {
  name        = "widgets.cogs"
  instance_id = ibm_resource_instance.dns.guid
  description = "this is a description"
  label       = "this-is-a-label"
}

resource "ibm_dns_permitted_network" "cloud" {
  instance_id = ibm_resource_instance.dns.guid
  zone_id     = ibm_dns_zone.widgets_cogs.zone_id
  vpc_crn     = ibm_is_vpc.cloud.crn
  type        = "vpc"
}

resource "ibm_dns_glb_monitor" "cloud" {
  depends_on  = [ibm_dns_zone.widgets_cogs]
  name        = var.prefix
  instance_id = ibm_resource_instance.dns.guid
  description = "ibm_dns_glb_monitor cloud description"
  interval    = 60
  retries     = 2
  # timeout        = default is good
  port           = 80
  type           = "HTTP"
  expected_codes = "200"
  path           = "/"
  method         = "GET"
}
// configure custom resolvers with a minimum of two resolver locations.
resource "ibm_dns_custom_resolver" "cloud" {
  name        = var.prefix
  instance_id = ibm_resource_instance.dns.guid
  description = "onprem uses this resolver to find the glbs and zones"
  enabled     = true
  dynamic "locations" {
    for_each = module.zone
    content {
      subnet_crn = locations.value.subnet_dns.resource_crn
      enabled    = true
    }
  }
}