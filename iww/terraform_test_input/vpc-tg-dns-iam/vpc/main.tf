resource "ibm_is_vpc" "vpc" {
  name                      = var.basename
  resource_group            = var.resource_group_id
  address_prefix_management = "manual"
}

resource "ibm_is_vpc_address_prefix" architecture {
  for_each = var.vpc_architecture.zones
  vpc      = ibm_is_vpc.vpc.id
  name     = "${var.basename}-${each.key}"
  zone     = "${var.region}-${each.value.zone_id}"
  cidr     = each.value.cidr
}

resource "ibm_is_subnet" "subnets" {
  vpc             = ibm_is_vpc.vpc.id
  resource_group  = var.resource_group_id
  for_each        = var.vpc_architecture.zones
  name            = "${var.basename}-${each.key}"
  zone            = "${var.region}-${each.value.zone_id}"
  ipv4_cidr_block = each.value.cidr
  depends_on = [
    ibm_is_vpc_address_prefix.architecture
  ]
}
