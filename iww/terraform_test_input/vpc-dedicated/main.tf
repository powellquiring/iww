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


/****************************************


resource "tls_private_key" "ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

output "generated_ssh_key" {
  value     = tls_private_key.ssh
  sensitive = true
}

resource "local_file" "ssh-key" {
  content         = tls_private_key.ssh.private_key_pem
  filename        = "../generated_key_rsa"
  file_permission = "0600"
}

resource "ibm_is_ssh_key" "generated_key" {
  name           = "${var.basename}-${var.region}-key"
  public_key     = tls_private_key.ssh.public_key_openssh
  resource_group = data.ibm_resource_group.group.id
}


locals {
  unique_id       = "000"
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
  myip        = data.external.ifconfig_me.result.ip # replace with your IP if ifconfig.me does not work
  ssh_key_ids = var.ssh_key_name != "" ? [data.ibm_is_ssh_key.ssh_key[0].id, ibm_is_ssh_key.generated_key.id] : [ibm_is_ssh_key.generated_key.id]
  image_id    = data.ibm_is_image.name.id
}


locals {
  subnets_front = { for zone_number in range(var.subnets) : zone_number => {
    cidr = cidrsubnet(ibm_is_vpc_address_prefix.locations[zone_number].cidr, 8, 0) # need a dependency on address prefix
    zone = local.prefixes[zone_number].zone
  } }
  subnets_back = { for zone_number in range(var.subnets) : zone_number => {
    cidr = cidrsubnet(ibm_is_vpc_address_prefix.locations[zone_number].cidr, 8, 1) # need a dependency on address prefix
    zone = local.prefixes[zone_number].zone
  } }
}

resource "ibm_is_subnet" "front" {
  for_each        = local.subnets_front
  name            = "${var.basename}-front-${each.key}"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = each.value.zone
  ipv4_cidr_block = each.value.cidr
}

resource "ibm_is_subnet" "back" {
  for_each        = local.subnets_back
  name            = "${var.basename}-back-${each.key}"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = each.value.zone
  ipv4_cidr_block = each.value.cidr
}

resource "ibm_is_lb" "front" {
  name           = "${local.name}-front"
  subnets        = [for subnet in ibm_is_subnet.front : subnet.id]
  type           = "public"
  resource_group = local.resource_group
}

resource "ibm_is_lb" "back" {
  name           = "${local.name}-back"
  subnets        = [for subnet in ibm_is_subnet.back : subnet.id]
  type           = "private"
  resource_group = local.resource_group
}

resource "ibm_is_security_group_rule" "inbound_myip" {
  group     = ibm_is_vpc.location.default_security_group
  direction = "inbound"
  # remote    = local.myip
  tcp {
    port_min = 22
    port_max = 22
  }
}

resource "ibm_is_security_group_rule" "inbound_8000" {
  group     = ibm_is_vpc.location.default_security_group
  direction = "inbound"
  tcp {
    port_min = 8000
    port_max = 8000
  }
}

locals {
  user_data0 = file("${path.module}/user_data.sh")
  user_data  = replace(replace(local.user_data0, "__MAIN_PY__", file("${path.module}/../app/main.py")), "__POSTGRESQL_PY__", file("${path.module}/../app/postgresql.py"))
}

#################### front ################
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
  user_data = replace(replace(local.user_data, "__FRONT_BACK__", "front"), "__REMOTE_URL__", "http://${ibm_is_lb.back.hostname}:8000")
  tags      = local.tags
}

resource "ibm_is_lb_pool" "front" {
  lb                  = ibm_is_lb.front.id
  name                = "front"
  protocol            = "http"
  algorithm           = "round_robin"
  health_delay        = "5"
  health_retries      = "2"
  health_timeout      = "2"
  health_type         = "http"
  health_monitor_url  = "/health"
  health_monitor_port = "8000"
}

resource "ibm_is_lb_listener" "front" {
  lb           = ibm_is_lb.front.id
  port         = "8000"
  protocol     = "http"
  default_pool = ibm_is_lb_pool.front.id
}

resource "ibm_is_lb_pool_member" "front" {
  for_each       = ibm_is_instance.front
  lb             = ibm_is_lb.front.id
  pool           = element(split("/", ibm_is_lb_pool.front.id), 1)
  port           = "8000"
  target_address = each.value.primary_network_interface[0].primary_ipv4_address
}

output "lb_front" {
  value = "http://${ibm_is_lb.front.hostname}:8000"
}

resource "ibm_is_floating_ip" "front" {
  for_each       = ibm_is_instance.front
  name           = each.value.name
  target         = each.value.primary_network_interface[0].id
  resource_group = local.resource_group
  tags           = local.tags
}

output "instances_front" {
  value = { for key, instance in ibm_is_instance.front : key => {
    name                 = instance.name
    primary_ipv4_address = instance.primary_network_interface[0].primary_ipv4_address
    fip                  = ibm_is_floating_ip.front[key].address
  } }
}


#################### back ################
locals {
  postgresql_credentials = jsonencode(nonsensitive(ibm_resource_key.postgresql.credentials))
  user_data_back         = replace(replace(replace(local.user_data, "__FRONT_BACK__", "back"), "__REMOTE_URL__", ""), "__POSTGRESQL_CREDENTIALS__", local.postgresql_credentials)
}

resource "local_file" "postgresql" {
  content  = local.postgresql_credentials
  filename = "${path.module}/../app/terraform_service_credentials.json"
}

resource "ibm_is_instance" "back" {
  for_each       = ibm_is_subnet.back
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
  user_data = local.user_data_back
  tags      = local.tags
}

resource "ibm_is_lb_pool" "back" {
  lb                  = ibm_is_lb.back.id
  name                = "back"
  protocol            = "http"
  algorithm           = "round_robin"
  health_delay        = "5"
  health_retries      = "2"
  health_timeout      = "2"
  health_type         = "http"
  health_monitor_url  = "/health"
  health_monitor_port = "8000"
}

resource "ibm_is_lb_listener" "back" {
  lb           = ibm_is_lb.back.id
  port         = "8000"
  protocol     = "http"
  default_pool = ibm_is_lb_pool.back.id
}

resource "ibm_is_lb_pool_member" "back" {
  for_each       = ibm_is_instance.back
  lb             = ibm_is_lb.back.id
  pool           = element(split("/", ibm_is_lb_pool.back.id), 1)
  port           = "8000"
  target_address = each.value.primary_network_interface[0].primary_ipv4_address
}

output "lb_back" {
  value = "http://${ibm_is_lb.back.hostname}:8000"
}

resource "ibm_is_floating_ip" "back" {
  for_each       = ibm_is_instance.back
  name           = each.value.name
  target         = each.value.primary_network_interface[0].id
  resource_group = local.resource_group
  tags           = local.tags
}

output "instances_back" {
  value = { for key, instance in ibm_is_instance.back : key => {
    name                 = instance.name
    primary_ipv4_address = instance.primary_network_interface[0].primary_ipv4_address
    fip                  = ibm_is_floating_ip.back[key].address
  } }
}

***************************************/
