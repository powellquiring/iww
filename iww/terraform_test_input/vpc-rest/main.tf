#### Data
data "ibm_resource_group" "group" {
  name = var.resource_group_name
}
data "ibm_is_image" "name" {
  name = var.image_name
}

data "ibm_is_ssh_key" "ssh_key" {
  count = var.ssh_key_name != "" ? 1 : 0
  name  = var.ssh_key_name
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
    cidr = cidrsubnet(ibm_is_vpc_address_prefix.locations.cidr, 8, 0) # need a dependency on address prefix
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
  name = "${local.name}-0"
  zone = local.prefixes[0].zone
  vpc  = ibm_is_vpc.location.id
  cidr = local.prefixes[0].cidr
}


#################### front ################
resource "ibm_is_subnet" "front" {
  #for_each        = local.subnets_front
  name            = "${var.basename}-front-0"
  resource_group  = local.resource_group
  vpc             = ibm_is_vpc.location.id
  zone            = local.subnets_front[0].zone
  ipv4_cidr_block = local.subnets_front[0].cidr
}

resource "ibm_is_instance" "front" {
  #for_each       = ibm_is_subnet.front
  name           = ibm_is_subnet.front.name
  vpc            = ibm_is_subnet.front.vpc
  zone           = ibm_is_subnet.front.zone
  keys           = local.ssh_key_ids
  image          = local.image_id
  profile        = var.profile
  resource_group = local.resource_group
  primary_network_interface {
    subnet = ibm_is_subnet.front.id
  }
  tags = local.tags
}

resource "ibm_is_snapshot" "frontsnap" {
  name           = ibm_is_instance.front.name
  source_volume  = ibm_is_instance.front.volume_attachments[0].volume_id
  resource_group = local.resource_group
}


##################### rest
/** TODO
resource "ibm_is_instance_network_interface" "main" {
  instance = ibm_is_instance.front.id
  subnet   = ibm_is_subnet.front.id
  #allow_ip_spoofing    = false
  name                 = local.name
  primary_ipv4_address = "10.0.0.5"

}
**/

resource "ibm_is_ike_policy" "main" {
  resource_group           = local.resource_group
  name                     = local.name
  authentication_algorithm = "md5"
  encryption_algorithm     = "triple_des"
  dh_group                 = 2
  ike_version              = 1
}

/*
todo is this needed?  snapshots not working if stopped
resource "ibm_is_instance_action" "main" {
  action       = "stop"
  force_action = false
  instance     = ibm_is_instance.front.id
}
*/

resource "ibm_is_instance_disk_management" "main" {
  instance = ibm_is_instance.front.id
  disks {
    name = local.name
    id   = ibm_is_instance.front.disks.0.id
    # id   = data.ibm_is_instance.ds_instance.disks.0.id
  }
}
/*** TODO
**** TODO  */

resource "ibm_is_instance_template" "main" {
  resource_group = local.resource_group
  name           = local.name
  image          = "r006-14140f94-fcc4-11e9-96e7-a72723715315"
  profile        = var.profile

  vpc  = ibm_is_subnet.front.vpc
  zone = ibm_is_subnet.front.zone
  keys = local.ssh_key_ids

  primary_network_interface {
    subnet = ibm_is_subnet.front.id
    #allow_ip_spoofing = false
  }

}
resource "ibm_is_instance_group" "main" {
  resource_group    = local.resource_group
  name              = local.name
  instance_template = ibm_is_instance_template.main.id
  instance_count    = 2
  subnets           = [ibm_is_subnet.front.id]
}
resource "ibm_is_instance_group_manager" "main" {
  name                 = local.name
  instance_group       = ibm_is_instance_group.main.id
  enable_manager       = true
  manager_type         = "scheduled"
  max_membership_count = 2
  min_membership_count = 1
}
resource "ibm_is_instance_group_manager_action" "main" {
  name                   = local.name
  instance_group         = ibm_is_instance_group.main.id
  instance_group_manager = ibm_is_instance_group_manager.main.manager_id
  cron_spec              = "*/5 1,2,3 * * *"
  membership_count       = 1
}

/* TODO ------------------------------------------------------------------------
resource "ibm_is_instance_group_membership" "main" {
  instance_group            = ibm_is_instance_group.main.id
  instance_group_membership = ibm_is_instance.front.id
  name                      = local.name
}
**/


resource "ibm_is_instance_group" "autoscale" {
  name              = "${local.name}-autoscale"
  resource_group    = local.resource_group
  instance_template = ibm_is_instance_template.main.id
  instance_count    = 2
  subnets           = [ibm_is_subnet.front.id]
}

resource "ibm_is_instance_group_manager" "autoscale" {
  name                 = "${local.name}-autoscale"
  aggregation_window   = 120
  instance_group       = ibm_is_instance_group.autoscale.id
  cooldown             = 300
  manager_type         = "autoscale"
  enable_manager       = true
  max_membership_count = 2
  min_membership_count = 1
}


resource "ibm_is_instance_group_manager_policy" "autoscale" {
  name                   = "${local.name}-autoscale"
  instance_group         = ibm_is_instance_group.autoscale.id
  instance_group_manager = ibm_is_instance_group_manager.autoscale.manager_id
  metric_type            = "cpu"
  metric_value           = 70
  policy_type            = "target"
}

/* TODO ------------------------------------------------------------------------
-------------------------------------------------------------------------------------**/
/** TODO
**/
resource "ibm_is_volume" "main" {
  resource_group = local.resource_group
  name           = local.name
  profile        = "10iops-tier"
  zone           = "us-south-1"

}
resource "ibm_is_vpc_route" "main" {
  name        = local.name
  vpc         = ibm_is_subnet.front.vpc
  zone        = "us-south-1"
  destination = "192.168.4.0/24"
  next_hop    = "10.0.0.4"

}
resource "ibm_is_vpc_routing_table" "main" {
  vpc                           = ibm_is_subnet.front.vpc
  name                          = local.name
  route_direct_link_ingress     = true
  route_transit_gateway_ingress = false
  route_vpc_zone_ingress        = false

}
resource "ibm_is_vpn_gateway" "main" {
  resource_group = local.resource_group
  name           = local.name
  subnet         = ibm_is_subnet.front.id
  mode           = "route"

}
resource "ibm_is_vpn_gateway_connection" "main" {
  name          = local.name
  vpn_gateway   = ibm_is_vpn_gateway.main.id
  peer_address  = ibm_is_vpn_gateway.main.public_ip_address
  preshared_key = "VPNDemoPassword"
  local_cidrs   = [ibm_is_subnet.front.ipv4_cidr_block]
  peer_cidrs    = ["10.1.0.0/8"]
}
resource "ibm_is_vpc_routing_table_route" "main" {
  vpc           = ibm_is_subnet.front.vpc
  routing_table = ibm_is_vpc_routing_table.main.routing_table
  zone          = "us-south-1"
  name          = local.name
  destination   = "192.168.4.0/24"
  action        = "deliver"
  # next_hop      = ibm_is_vpn_gateway_connection.main.gateway_connection // Example value "10.0.0.4"
  next_hop = "1.1.1.1"

}

/*************
  boot_volume {
    name                             = "testbootvol"
    delete_volume_on_instance_delete = true
  }
  volume_attachments {
    delete_volume_on_instance_delete = true
    name                             = "volatt-01"
    volume_prototype {
      iops     = 3000
      profile  = "general-purpose"
      capacity = 200
    }

  }
  resource "ibm_is_instance_volume_attachment" "main" {
    instance = ibm_is_instance.testacc_instance.id

    name                               = "test-vol-att-1"
    profile                            = "general-purpose"
    capacity                           = "20"
    delete_volume_on_attachment_delete = true
    delete_volume_on_instance_delete   = true
    volume_name                        = "testvol1"

    //User can configure timeouts
    timeouts {
      create = "15m"
      update = "15m"
      delete = "15m"
    }

  }
  resource "ibm_is_ipsec_policy" "main" {
    name                     = "test"
    authentication_algorithm = "md5"
    encryption_algorithm     = "triple_des"
    pfs                      = "disabled"

  }
  resource "ibm_is_placement_group" "main" {
    strategy = "host_spread"
    name     = "my-placement-group"

  }
  resource "ibm_is_snapshot" "main" {
    name          = "testsnapshot"
    source_volume = ibm_is_instance.testacc_instance.volume_attachments[0].volume_id

    //User can configure timeouts
    timeouts {
      create = "15m"
      delete = "15m"
    }

  }
  resource "ibm_is_virtual_endpoint_gateway" "main" {
    name = "my-endpoint-gateway-1"
    target {
      name          = "ibm-ntp-server"
      resource_type = "provider_infrastructure_service"
    }
  vpc            = ibm_is_subnet.front.vpc
    ips {
      id = "0737-5ab3c18e-6f6c-4a69-8f48-20e3456647b5"
    }
    resource_group = data.ibm_resource_group.test_acc.id

  }
  resource "ibm_is_virtual_endpoint_gateway_ip" "main" {
    gateway     = ibm_is_virtual_endpoint_gateway.endpoint_gateway.id
    reserved_ip = "0737-5ab3c18e-6f6c-4a69-8f48-20e3456647b5"
  }
*************/
