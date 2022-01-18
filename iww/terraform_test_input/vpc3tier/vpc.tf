data "ibm_is_image" "name" {
  name = var.image_name
}

data "external" "ifconfig_me" {
  program = ["bash", "-c", <<-EOS
    echo '{"ip": "'$(curl ifconfig.me)'"}'
  EOS
  ]
}

data "ibm_resource_group" "group" {
  name = var.resource_group_name
}

data "ibm_is_ssh_key" "ssh_key" {
  count = var.ssh_key_name != "" ? 1 : 0
  name = var.ssh_key_name
}

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
    lower(replace("dir:${abspath(path.root)}", "/", "_")),
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

# This is the first attempt at locking down  more tightly based on security groups
# Now the default security group can be removed from some of the resources
# - back end load_balancer - now
# - instances - No, they need private dns access on port 53 ip addresses 161.26.0.10/11
# - front end load balancer - no, it needs to allow port 8000 from anywhere
resource ibm_is_security_group load_balancer_targets {
  name     = "${local.name}-load-balancer-targets"
  vpc      = ibm_is_vpc.location.id
  resource_group = data.ibm_resource_group.group.id
}

# the load balancer has outbound traffic to targets listening on port 8000
# the front end target has outbound traffic to the backend load balancer on port 8000
# (backend targets do not need outbound to 8000)
resource "ibm_is_security_group_rule" "load_balancer_targets_outbound" {
  direction = "outbound"
  group = ibm_is_security_group.load_balancer_targets.id
  remote = ibm_is_security_group.load_balancer_targets.id
  tcp {
    port_min = 8000
    port_max = 8000
  }
}

# the targets will receive input from their load load_balancers on port 8000
# the backend load balancer will receive input from the front end targets on port 8000
# (the front end load balancer will not need inbound traffic from this security group)
resource "ibm_is_security_group_rule" "load_balancer_targets_inbound" {
  direction = "inbound"
  group = ibm_is_security_group.load_balancer_targets.id
  remote = ibm_is_security_group.load_balancer_targets.id
  tcp {
    port_min = 8000
    port_max = 8000
  }
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
  public_gateways = { for zone_number in range(var.subnets) : zone_number => {
    zone = local.prefixes[zone_number].zone
  } }
}

resource "ibm_is_public_gateway" "zone" {
  for_each        = local.public_gateways
  name = "${var.basename}-${each.value.zone}"
  vpc             = ibm_is_vpc.location.id
  zone = each.value.zone
  resource_group  = local.resource_group
}


resource "ibm_is_subnet" "front" {
  for_each        = local.subnets_front
  name            = "${var.basename}-front-${each.key}"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = each.value.zone
  ipv4_cidr_block = each.value.cidr
  public_gateway = ibm_is_public_gateway.zone[each.key].id
}

resource "ibm_is_subnet" "back" {
  for_each        = local.subnets_back
  name            = "${var.basename}-back-${each.key}"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = each.value.zone
  ipv4_cidr_block = each.value.cidr
  public_gateway = ibm_is_public_gateway.zone[each.key].id
}

resource "ibm_is_lb" "front" {
  name           = "${local.name}-front"
  subnets        = [for subnet in ibm_is_subnet.front : subnet.id]
  type           = "public"
  resource_group = local.resource_group
}
// TODO add these
resource "ibm_is_lb_listener_policy" "main" {}
resource "ibm_is_lb_listener_policy_rule" "main" {}

/*######################################################################
# This causes a problem when resources are destroyed
# This breaks automated testing
# https://github.com/IBM-Cloud/terraform-provider-ibm/issues/3204
resource ibm_is_security_group_target load_balancer_targets_front {
  security_group = ibm_is_security_group.load_balancer_targets.id
  target = ibm_is_lb.front.id
}
######################################################################*/


resource "ibm_is_lb" "back" {
  name           = "${local.name}-back"
  subnets        = [for subnet in ibm_is_subnet.back : subnet.id]
  type           = "private"
  resource_group = local.resource_group
}

/*######################################################################
# This causes a problem when resources are destroyed
# This breaks automated testing
# https://github.com/IBM-Cloud/terraform-provider-ibm/issues/3204
resource ibm_is_security_group_target load_balancer_targets_back {
  security_group = ibm_is_security_group.load_balancer_targets.id
  target = ibm_is_lb.back.id
}
######################################################################*/

resource "ibm_is_security_group_rule" "inbound_8000" {
  group     = ibm_is_vpc.location.default_security_group
  direction = "inbound"
  tcp {
    port_min = 8000
    port_max = 8000
  }
}
resource "ibm_is_security_group_rule" "outbound_8000" {
  group     = ibm_is_vpc.location.default_security_group
  direction = "outbound"
  tcp {
    port_min = 8000
    port_max = 8000
  }
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

locals {
  user_data0 = file("${path.module}/user_data.sh")
  bash_variables = <<EOS
LOGDNA_INGESTION_KEY=${local.logdna_ingestion_key}
SYSDIG_INGESTION_KEY=${local.sysdig_ingestion_key}
POSTGRESQL=${var.postgresql}
REGION=${var.region}
EOS
  user_data  = replace(replace(replace(local.user_data0, "__MAIN_PY__", file("${path.module}/app/main.py")), "__POSTGRESQL_PY__", file("${path.module}/app/postgresql.py")), "__BASH_VARIABLES__", local.bash_variables)
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

resource ibm_is_security_group_target load_balancer_targets_instances_front {
  for_each       = ibm_is_instance.front
  security_group = ibm_is_security_group.load_balancer_targets.id
  target = each.value.primary_network_interface[0].id
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
  user_data_back         = replace(replace(replace(local.user_data, "__FRONT_BACK__", "back"), "__REMOTE_URL__", ""), "__POSTGRESQL_CREDENTIALS__", local.postgresql_credentials)
}

resource "local_file" "postgresql" {
  content  = local.postgresql_credentials
  filename = "${path.module}/app/terraform_service_credentials.json"
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

resource ibm_is_security_group_target load_balancer_targets_instances_back {
  for_each       = ibm_is_instance.back
  security_group = ibm_is_security_group.load_balancer_targets.id
  target = each.value.primary_network_interface[0].id
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
