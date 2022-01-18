variable basename {}
variable vpc {}
variable cidr_remote {}
variable resource_group {}

resource "ibm_is_security_group" "ssh" {
  name           = "${var.basename}-ssh"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}
resource "ibm_is_security_group_rule" "ssh" {
  group     = ibm_is_security_group.ssh.id
  direction = "inbound"
  tcp {
    port_min = 22
    port_max = 22
  }
}


#-----------------------------------------------------
resource "ibm_is_security_group" "install_software" {
  name           = "${var.basename}-install-software"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}

resource "ibm_is_security_group_rule" "egress_443_all" {
  group     = ibm_is_security_group.install_software.id
  direction = "outbound"
  remote    = "161.26.0.6"
  tcp {
    port_min = 443
    port_max = 443
  }
}

resource "ibm_is_security_group_rule" "egress_80_all" {
  group     = ibm_is_security_group.install_software.id
  direction = "outbound"
  remote    = "161.26.0.6"
  tcp {
    port_min = 80
    port_max = 80
  }
}

resource "ibm_is_security_group_rule" "egress_dns_udp_10" {
  group     = ibm_is_security_group.install_software.id
  direction = "outbound"
  remote    = "161.26.0.10"
  udp {
    port_min = 53
    port_max = 53
  }
}

resource "ibm_is_security_group_rule" "egress_dns_udp_11" {
  group     = ibm_is_security_group.install_software.id
  direction = "outbound"
  remote    = "161.26.0.11"
  udp {
    port_min = 53
    port_max = 53
  }
}


#-----------------------------------------------------
resource "ibm_is_security_group" "data_inbound" {
  name           = "${var.basename}-data-inbound"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}

resource "ibm_is_security_group_rule" "shared_ingress_all" {
  group     = ibm_is_security_group.data_inbound.id
  direction = "inbound"
  remote    = var.cidr_remote
  tcp {
    port_min = 3000
    port_max = 3000
  }
}

#-----------------------------------------------------
resource "ibm_is_security_group" "data_inbound_insecure" {
  name           = "${var.basename}-datain-insecure"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}

resource "ibm_is_security_group_rule" "shared_ingress_all_insecure" {
  group     = ibm_is_security_group.data_inbound_insecure.id
  direction = "inbound"
  # remote    = var.cidr_remote insecure, in from my laptop
  tcp {
    port_min = 3000
    port_max = 3000
  }
}


#-----------------------------------------------------
# vsi1/app needs access to vsi2/app
resource "ibm_is_security_group" "data_outbound" {
  name           = "${var.basename}-data-outbound"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}

resource "ibm_is_security_group_rule" "application_egress_app_all" {
  group     = ibm_is_security_group.data_outbound.id
  direction = "outbound"
  remote    = var.cidr_remote
  tcp {
    port_min = 3000
    port_max = 3000
  }
}

#-----------------------------------------------------
# inbound/outbound data connected to each other
#-----------------------------------------------------
resource "ibm_is_security_group" "data_inbound_from_outbound" {
  name           = "${var.basename}-data-inbound-from-outbound"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}
resource "ibm_is_security_group_rule" "data-inbound-from-outbound" {
  group     = ibm_is_security_group.data_inbound_from_outbound.id
  direction = "inbound"
  remote    = ibm_is_security_group.data_outbound_to_inbound.id
  tcp {
    port_min = 3000
    port_max = 3000
  }
}
#-----------------------------------------------------
resource "ibm_is_security_group" "data_outbound_to_inbound" {
  name           = "${var.basename}-data-outbound-to-inbound"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}
resource "ibm_is_security_group_rule" "data-outbound-to-inbound" {
  group     = ibm_is_security_group.data_outbound_to_inbound.id
  direction = "outbound"
  remote    = ibm_is_security_group.data_inbound_from_outbound.id
  tcp {
    port_min = 3000
    port_max = 3000
  }
}

#-----------------------------------------------------
resource "ibm_is_security_group" "ibm_dns" {
  name           = "${var.basename}-ibm-dns"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}
resource "ibm_is_security_group_rule" "outbound_dns_udp_7" {
  group     = ibm_is_security_group.ibm_dns.id
  direction = "outbound"
  remote    = "161.26.0.7"
  udp {
    port_min = 53
    port_max = 53
  }
}
resource "ibm_is_security_group_rule" "outbound_dns_udp_8" {
  group     = ibm_is_security_group.ibm_dns.id
  direction = "outbound"
  remote    = "161.26.0.8"
  udp {
    port_min = 53
    port_max = 53
  }
}

#-----------------------------------------------------
resource "ibm_is_security_group" "outbound_all" {
  name           = "${var.basename}-outbound-all"
  resource_group = var.resource_group.id
  vpc            = var.vpc.id
}
resource "ibm_is_security_group_rule" "outbound_all" {
  group     = ibm_is_security_group.outbound_all.id
  direction = "outbound"
  #remote    = ALL
  # udp ALL
  # tcp ALL
}

#-----------------------------------------------------
output security_group_ssh {
  value = ibm_is_security_group.ssh
}
output security_group_install_software {
  value = ibm_is_security_group.install_software
}
output security_group_data_inbound {
  value = ibm_is_security_group.data_inbound
}
output security_group_data_inbound_insecure {
  value = ibm_is_security_group.data_inbound_insecure
}
output security_group_data_outbound {
  value = ibm_is_security_group.data_outbound
}
output security_group_ibm_dns {
  value = ibm_is_security_group.ibm_dns
}
output security_group_outbound_all {
  value = ibm_is_security_group.outbound_all
}
output security_group_data_outbound_to_inbound {
  value = ibm_is_security_group.data_outbound_to_inbound
}
output security_group_data_inbound_from_outbound {
  value = ibm_is_security_group.data_inbound_from_outbound
}
