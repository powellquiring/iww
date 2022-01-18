provider "ibm" {
  region           = var.region
  ibmcloud_api_key = var.ibmcloud_api_key
}

data "ibm_resource_group" "network" {
  name = "${var.resource_group_name}"
}

data "ibm_resource_group" "shared" {
  name = "${var.resource_group_name}"
}
data "ibm_resource_group" "application1" {
  name = "${var.resource_group_name}"
}
data "ibm_resource_group" "application2" {
  name = "${var.resource_group_name}"
}

#-------------------------------------------------------------------
module "vpc_shared" {
  source            = "./vpc"
  region            = var.region
  vpc_architecture  = var.network_architecture.shared
  basename          = "${var.basename}-shared"
  resource_group_id = data.ibm_resource_group.shared.id
}
module "sg_shared" {
  source         = "./sg"
  basename       = "${var.basename}-sg-shared"
  vpc            = module.vpc_shared.vpc
  resource_group = data.ibm_resource_group.shared
  cidr_remote    = var.network_architecture.shared.cidr_remote
}

module "vpc_application1" {
  source            = "./vpc"
  region            = var.region
  vpc_architecture  = var.network_architecture.application1
  basename          = "${var.basename}-application1"
  resource_group_id = data.ibm_resource_group.application1.id
}
module "sg_application1" {
  source         = "./sg"
  basename       = "${var.basename}-app1"
  vpc            = module.vpc_application1.vpc
  resource_group = data.ibm_resource_group.application1
  cidr_remote    = var.network_architecture.application1.cidr_remote
}

module "vpc_application2" {
  source            = "./vpc"
  region            = var.region
  vpc_architecture  = var.network_architecture.application2
  basename          = "${var.basename}-application2"
  resource_group_id = data.ibm_resource_group.application2.id
}
module "sg_application2" {
  source         = "./sg"
  basename       = "${var.basename}-app2"
  vpc            = module.vpc_application2.vpc
  resource_group = data.ibm_resource_group.application2
  cidr_remote    = var.network_architecture.application2.cidr_remote
}

#-------------------------------------------------------------------
resource "ibm_resource_instance" "dns" {
  name              = "${var.basename}-dns"
  resource_group_id = data.ibm_resource_group.network.id
  location          = "global"
  service           = "dns-svcs"
  plan              = "standard-dns"
}

resource "ibm_dns_zone" "widgets_com" {
  name        = "widgets.com"
  instance_id = ibm_resource_instance.dns.guid
  description = "this is a description"
  label       = "this-is-a-label"
}

resource "ibm_dns_permitted_network" "shared" {
  instance_id = ibm_resource_instance.dns.guid
  zone_id     = ibm_dns_zone.widgets_com.zone_id
  vpc_crn     = module.vpc_shared.vpc.crn
  type        = "vpc"
}

resource "ibm_dns_permitted_network" "application1" {
  depends_on  = [ibm_dns_permitted_network.shared]
  instance_id = ibm_resource_instance.dns.guid
  zone_id     = ibm_dns_zone.widgets_com.zone_id
  vpc_crn     = module.vpc_application1.vpc.crn
  type        = "vpc"
}

resource "ibm_dns_permitted_network" "application2" {
  depends_on  = [ibm_dns_permitted_network.application1]
  instance_id = ibm_resource_instance.dns.guid
  zone_id     = ibm_dns_zone.widgets_com.zone_id
  vpc_crn     = module.vpc_application2.vpc.crn
  type        = "vpc"
}

resource "ibm_tg_gateway" "tgw" {
  count          = var.transit_gateway ? 1 : 0
  name           = "${var.basename}-tgw"
  location       = var.region
  global         = false
  resource_group = data.ibm_resource_group.network.id
}

resource "ibm_tg_connection" "shared" {
  count        = var.transit_gateway ? 1 : 0
  network_type = "vpc"
  gateway      = ibm_tg_gateway.tgw[0].id
  name         = "${var.basename}-shared"
  network_id   = module.vpc_shared.vpc.resource_crn
}
resource "ibm_tg_connection" "application1" {
  count        = var.transit_gateway ? 1 : 0
  network_type = "vpc"
  gateway      = ibm_tg_gateway.tgw[0].id
  name         = "${var.basename}-application1"
  network_id   = module.vpc_application1.vpc.resource_crn
}
resource "ibm_tg_connection" "application2" {
  count        = var.transit_gateway ? 1 : 0
  network_type = "vpc"
  gateway      = ibm_tg_gateway.tgw[0].id
  name         = "${var.basename}-application2"
  network_id   = module.vpc_application2.vpc.resource_crn
}
