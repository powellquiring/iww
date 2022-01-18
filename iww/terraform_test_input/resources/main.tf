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

resource "ibm_resource_instance" "logdna" {
  name              = "${local.name}-logdna"
  service           = "logdna"
  plan              = "7-day"
  location          = var.region
  resource_group_id = local.resource_group
  tags              = local.tags
}

resource "ibm_resource_instance" "kms" {
  name = "${local.name}-kms"
  resource_group_id = local.resource_group
  service = "kms"
  plan = "tiered-pricing"
  location = var.region
}

resource "ibm_kms_key" "image_key" {
  count = 10
  instance_id  = ibm_resource_instance.kms.guid
  key_name     = "image_key-${count.index}"
  standard_key = false
  force_delete = true
}