# one origin pool for each zone (each nlb), with a health check from same zone
resource "ibm_dns_glb_pool" "cloud" {
  for_each                  = module.zone
  depends_on                = [ibm_dns_zone.widgets_cogs]
  name                      = each.value.name
  instance_id               = ibm_resource_instance.dns.guid
  enabled                   = true
  healthy_origins_threshold = 1
  origins {
    name    = each.value.zone
    address = "iwwtest.suite.cogs"
    enabled = true
  }
  monitor             = ibm_dns_glb_monitor.cloud.monitor_id
  healthcheck_region  = var.region
  healthcheck_subnets = [each.value.subnet_dns.resource_crn]
}

resource "ibm_dns_glb_pool" "test" {
  for_each                  = ibm_is_instance.test
  depends_on                = [ibm_dns_zone.widgets_cogs]
  name                      = each.value.name
  instance_id               = ibm_resource_instance.dns.guid
  enabled                   = true
  healthy_origins_threshold = 1
  origins {
    name    = each.value.name
    address = each.value.primary_network_interface[0].primary_ipv4_address
    enabled = true
  }
  monitor             = ibm_dns_glb_monitor.cloud.monitor_id
  healthcheck_region  = var.region
  healthcheck_subnets = [ibm_is_subnet.test.resource_crn]
}

resource "ibm_dns_glb" "widgets" {
  name          = "backend"
  enabled       = true
  instance_id   = ibm_resource_instance.dns.guid
  zone_id       = ibm_dns_zone.widgets_cogs.zone_id
  description   = "ibm_dns_glb description"
  ttl           = 120
  default_pools = [for key, pool in ibm_dns_glb_pool.cloud : pool.pool_id]
  fallback_pool = ibm_dns_glb_pool.test[1].pool_id
  dynamic "az_pools" {
    for_each = module.zone
    content {
      availability_zone = az_pools.value.zone
      pools             = [ibm_dns_glb_pool.cloud[az_pools.key].pool_id]
    }
  }
}

resource "ibm_is_vpc_address_prefix" "test" {
  for_each = local.test_zones
  vpc      = ibm_is_vpc.cloud.id
  name     = "${each.value.zone}-test"
  zone     = each.value.zone
  cidr     = each.value.cidr
}

locals {
  test = {
    name      = "${var.prefix}-test"
    zone      = ibm_is_vpc_address_prefix.test[0].zone
    cidr      = ibm_is_vpc_address_prefix.test[0].cidr
    user_data = local.user_data
    instances = 2
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
}

resource "ibm_is_subnet" "test" {
  name            = local.test.name
  tags            = local.tags
  resource_group  = data.ibm_resource_group.all_rg.id
  vpc             = local.test.vpc.id
  zone            = local.test.zone
  ipv4_cidr_block = local.test.cidr
}

resource "ibm_is_security_group" "test" {
  name           = "${local.test.name}-zone"
  vpc            = local.test.vpc.id
  resource_group = local.test.resource_group.id
}

resource "ibm_is_security_group_rule" "test_inbound_all" {
  group     = ibm_is_security_group.test.id
  direction = "inbound"
}
resource "ibm_is_security_group_rule" "test_outbound_all" {
  group     = ibm_is_security_group.test.id
  direction = "outbound"
}
resource "ibm_is_instance" "test" {
  for_each       = { for index in range(local.test.instances) : index => "${ibm_is_subnet.test.name}-${index}" }
  name           = each.value
  image          = local.test.image.id
  profile        = local.test.profile
  vpc            = local.test.vpc.id
  zone           = ibm_is_subnet.test.zone
  keys           = local.test.keys
  user_data      = <<-EOT
    ${local.test.user_data}
    echo ${each.value} > /var/www/html/instance
  EOT
  resource_group = local.test.resource_group.id

  primary_network_interface {
    subnet          = ibm_is_subnet.test.id
    security_groups = [ibm_is_security_group.test.id]
  }
}
resource "ibm_is_floating_ip" "zone" {
  for_each       = ibm_is_instance.test
  name           = each.value.name
  target         = each.value.primary_network_interface[0].id
  resource_group = local.test.resource_group.id
}