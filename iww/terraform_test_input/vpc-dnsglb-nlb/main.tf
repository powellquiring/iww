resource "ibm_is_vpc" "cloud" {
  name                      = var.prefix
  tags                      = local.tags
  resource_group            = data.ibm_resource_group.all_rg.id
  address_prefix_management = "manual"
}

resource "ibm_is_vpc_address_prefix" "cloud" {
  for_each = local.cidr_zones
  vpc      = ibm_is_vpc.cloud.id
  name     = each.value.zone
  zone     = each.value.zone
  cidr     = each.value.cidr
}

locals {
  user_data = <<-EOT
  #!/bin/bash
  set -x
  export DEBIAN_FRONTEND=noninteractive
  apt -qq -y update < /dev/null
  apt -qq -y install nginx npm < /dev/null
  EOT
}

module "zone" {
  for_each  = ibm_is_vpc_address_prefix.cloud
  source    = "./modules/zone"
  name      = "${var.prefix}-${each.value.zone}"
  zone      = each.value.zone
  cidr      = each.value.cidr
  user_data = local.user_data
  instances = local.instances
  profile   = local.profile
  keys      = [data.ibm_is_ssh_key.sshkey.id]
  vpc = {
    id = ibm_is_vpc.cloud.id
  }
  resource_group = {
    id = data.ibm_resource_group.all_rg.id
  }
  image = {
    id = data.ibm_is_image.os.id
  }
}