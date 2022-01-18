data "ibm_resource_group" "group" {
  name = var.resource_group_name
}

locals {
  provider_region = var.region
  name            = var.basename
  tags = [
    "basename:${var.basename}",
    replace("dir:${abspath(path.root)}", "/", "_"),
  ]
  resource_group = data.ibm_resource_group.group.id
}

resource "ibm_is_vpc" "location" {
  name                      = local.name
  resource_group            = local.resource_group
  address_prefix_management = "manual"
  tags                      = local.tags
}