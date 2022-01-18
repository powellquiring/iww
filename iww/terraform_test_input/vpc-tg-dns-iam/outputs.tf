output shared {
  value = {
    vpc                             = module.vpc_shared.vpc
    subnets                         = module.vpc_shared.subnets
    security_group_ssh              = module.sg_shared.security_group_ssh
    security_group_install_software = module.sg_shared.security_group_install_software
    security_group_data_inbound     = module.sg_shared.security_group_data_inbound
    security_group_data_outbound    = module.sg_shared.security_group_data_outbound
    security_group_data_outbound_to_inbound = module.sg_shared.security_group_data_outbound_to_inbound
    security_group_data_inbound_from_outbound = module.sg_shared.security_group_data_inbound_from_outbound
    security_group_ibm_dns          = module.sg_shared.security_group_ibm_dns
    security_group_outbound_all     = module.sg_shared.security_group_outbound_all
    dns = {
      guid    = ibm_resource_instance.dns.guid
      zone_id = ibm_dns_zone.widgets_com.zone_id
    }
  }
}

output application1 {
  value = {
    vpc                                  = module.vpc_application1.vpc
    subnets                              = module.vpc_application1.subnets
    security_group_ssh                   = module.sg_application1.security_group_ssh
    security_group_install_software      = module.sg_application1.security_group_install_software
    security_group_data_inbound          = module.sg_application1.security_group_data_inbound
    security_group_data_inbound_insecure = module.sg_application1.security_group_data_inbound_insecure
    security_group_data_outbound         = module.sg_application1.security_group_data_outbound
    security_group_ibm_dns               = module.sg_application1.security_group_ibm_dns
    security_group_outbound_all          = module.sg_application1.security_group_outbound_all
  }
}

output application2 {
  value = {
    vpc                                  = module.vpc_application2.vpc
    subnets                              = module.vpc_application2.subnets
    security_group_ssh                   = module.sg_application2.security_group_ssh
    security_group_install_software      = module.sg_application2.security_group_install_software
    security_group_data_inbound          = module.sg_application2.security_group_data_inbound
    security_group_data_inbound_insecure = module.sg_application2.security_group_data_inbound_insecure
    security_group_data_outbound         = module.sg_application2.security_group_data_outbound
    security_group_ibm_dns               = module.sg_application2.security_group_ibm_dns
    security_group_outbound_all          = module.sg_application2.security_group_outbound_all
  }
}
