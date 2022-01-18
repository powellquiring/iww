#### Data
data "ibm_resource_group" "group" {
  name = var.resource_group_name
}
data "ibm_is_image" "name" {
  name = var.image_name
}

data "ibm_is_ssh_key" "ssh_key" {
  count = var.ssh_key_name != "" ? 1 : 0
  name = var.ssh_key_name
}


#### Locals
locals {
  provider_region = var.region
  name            = var.basename
  tags = [
    "basename:${var.basename}",
    replace("dir:${abspath(path.root)}", "/", "_"),
  ]
  resource_group = data.ibm_resource_group.group.id
  cidr           = "10.0.0.0/8"
  prefixes = { for zone_number in range(3) : zone_number => {
    cidr = cidrsubnet(local.cidr, 8, zone_number)
    zone = "${var.region}-${zone_number + 1}"
  } }
  subnets_front = { for zone_number in range(var.subnets) : zone_number => {
    cidr = cidrsubnet(ibm_is_vpc_address_prefix.locations[zone_number].cidr, 8, 0) # need a dependency on address prefix
    zone = local.prefixes[zone_number].zone
  } }
  ssh_key_ids = [data.ibm_is_ssh_key.ssh_key[0].id]
  image_id    = data.ibm_is_image.name.id
}


#### Resources
resource "ibm_is_vpc" "location" {
  name                      = local.name
  resource_group            = local.resource_group
  address_prefix_management = "manual"
  tags                      = local.tags
}

resource "ibm_is_vpc_address_prefix" "locations" {
  for_each = local.prefixes
  name     = "${local.name}-${each.key}"
  zone     = each.value.zone
  vpc      = ibm_is_vpc.location.id
  cidr     = each.value.cidr
}


#################### front ################
resource "ibm_is_subnet" "front" {
  for_each        = local.subnets_front
  name            = "${var.basename}-front-${each.key}"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = each.value.zone
  ipv4_cidr_block = each.value.cidr
}

resource "ibm_is_instance" "front" {
  for_each       = ibm_is_subnet.front
  name           = each.value.name
  vpc            = each.value.vpc
  zone           = each.value.zone
  keys           = local.ssh_key_ids
  image          = local.image_id
  profile        = var.profile
  resource_group = local.resource_group
  primary_network_interface {
    subnet = each.value.id
  }
  tags      = local.tags
}
# TODO add these
resource "ibm_is_network_acl" "main" {
    name = "is-example-acl"
  vpc             = ibm_is_vpc.location.id
  rules {
    name        = "outbound"
    action      = "allow"
    source      = "0.0.0.0/0"
    destination = "0.0.0.0/0"
    direction   = "outbound"
    icmp {
      code = 1
      type = 1
    }
  }
  rules {
    name        = "inbound"
    action      = "allow"
    source      = "0.0.0.0/0"
    destination = "0.0.0.0/0"
    direction   = "inbound"
    icmp {
      code = 1
      type = 1
    }
  }

}
resource "ibm_is_network_acl_rule" "main" {
  network_acl    = ibm_is_network_acl.main.id
  name           = "inbound"
  action         = "allow"
  source         = "0.0.0.0/0"
  destination    = "0.0.0.0/0"
  direction      = "inbound"
}
resource "ibm_is_security_group_network_interface_attachment" "main" {
    security_group    = "2d364f0a-a870-42c3-a554-000001352417"
  network_interface = "6d6128aa-badc-45c4-bb0e-7c2c1c47be55"

}
resource "ibm_is_subnet_network_acl_attachment" "main" {
    subnet      = ibm_is_subnet.testacc_subnet.id
  network_acl = ibm_is_network_acl.isExampleACL.id

}
resource "ibm_is_subnet_reserved_ip" "main" {
          subnet = ibm_is_subnet.subnet1.id
        name = "my-subnet-reserved-ip1"
        target = ibm_is_virtual_endpoint_gateway.endpoint_gateway.id
}