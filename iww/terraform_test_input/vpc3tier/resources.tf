# TODO
# - sql query
resource "ibm_database" "postgresql" {
  count = var.postgresql ? 1 : 0
  name              = local.name
  resource_group_id = local.resource_group
  plan              = "standard"
  service           = "databases-for-postgresql"
  location          = var.region
  #service_endpoints = "private"
  tags = local.tags
}

resource "ibm_resource_key" "postgresql" {
  count = var.postgresql ? 1 : 0
  name                 = local.name
  resource_instance_id = ibm_database.postgresql[0].id
  # todo role?
  role = "Administrator"
  tags = local.tags
}
locals {
  postgresql_credentials = var.postgresql ? jsonencode(nonsensitive(ibm_resource_key.postgresql[0].credentials)) : "false"
  logdna = var.observability
  sysdig = var.observability
  flowlog = var.observability
}
output "postgresql" {
  value = var.postgresql
}


resource "ibm_resource_instance" "logdna" {
  count = local.logdna ? 1 : 0
  name              = local.name
  service           = "logdna"
  plan              = "7-day"
  location          = var.region
  resource_group_id = local.resource_group
  tags              = local.tags
}

resource "ibm_resource_key" "logdna" {
  count = local.logdna ? 1 : 0
  name              = local.name
  role                 = "Manager"
  resource_instance_id = ibm_resource_instance.logdna[0].id
  tags              = local.tags
}

locals {
  logdna_ingestion_key = local.logdna ? ibm_resource_key.logdna[0].credentials.ingestion_key : ""
  sysdig_ingestion_key = local.sysdig ? ibm_resource_key.sysdig[0].credentials["Sysdig Access Key"] : ""
}

resource "ibm_resource_instance" "sysdig" {
  count = local.sysdig ? 1 : 0
  name              = local.name
  service           = "sysdig-monitor"
  plan              = "graduated-tier"
  location          = var.region
  resource_group_id = local.resource_group
  tags              = local.tags
}

resource "ibm_resource_key" "sysdig" {
  count = local.sysdig ? 1 : 0
  name              = local.name
  role                 = "Manager"
  resource_instance_id = ibm_resource_instance.sysdig[0].id
  tags              = local.tags
}


## flow logs **
resource "ibm_resource_instance" "cos" {
  count = local.flowlog ? 1 : 0
  name              = local.name
  resource_group_id = local.resource_group
  service           = "cloud-object-storage"
  plan              = "standard"
  location          = "global"
  tags              = local.tags
}

resource "ibm_iam_authorization_policy" "is_flowlog_write_to_cos" {
  count = local.flowlog ? 1 : 0
  source_service_name  = "is"
  source_resource_type = "flow-log-collector"
  target_service_name  = "cloud-object-storage"
  target_resource_instance_id = ibm_resource_instance.cos[0].guid
  roles                = ["Writer"]
}

resource "ibm_cos_bucket" "flowlog" {
  count = local.flowlog ? 1 : 0
  bucket_name          = "${local.name}-flowlog-001"
  resource_instance_id = ibm_resource_instance.cos[0].id
  region_location      = var.region
  #storage_class        = "flex"
  storage_class        = "standard"
  force_delete         = true
}

resource ibm_is_flow_log test_flowlog {
  count = local.flowlog ? 1 : 0
  depends_on = [ibm_iam_authorization_policy.is_flowlog_write_to_cos[0]]
  name = local.name
  resource_group = local.resource_group
  target = ibm_is_vpc.location.id
  active = true
  storage_bucket = ibm_cos_bucket.flowlog[0].bucket_name
  tags              = local.tags
}

/*------------- is this used? TODO -------------------------------
output "postgresql" {
  value     = var.postgresql ? ibm_database.postgresql[0] : "false"
  sensitive = true
}
*/

/*--------------------------------------------------------------------------------
# TODO integrate with KMS
resource "ibm_resource_instance" "logdna_instance" {
  name              = local.name
  service           = "logdna"
  plan              = "7-day"
  location          = var.region
  resource_group_id = local.resource_group
  tags              = local.tags
}

resource "ibm_resource_key" "logdna" {
  name              = local.name
  role                 = "Manager"
  resource_instance_id = ibm_resource_instance.logdna_instance.id
  tags              = local.tags
}

# TODO integrate with KMS
resource "ibm_resource_instance" "secrets_manager" {
  name              = local.name
  service           = "secrets-manager"
  plan              = "lite"
  location          = var.region
  resource_group_id = local.resource_group
  tags              = local.tags
}

resource "ibm_resource_key" "secrets_manager" {
  name              = local.name
  role                 = "Manager"
  resource_instance_id = ibm_resource_instance.secrets_manager.id
  tags              = local.tags
}

resource "ibm_resource_instance" "cos" {
  name              = local.name
  resource_group_id = local.resource_group
  service           = "cloud-object-storage"
  plan              = "standard"
  location          = "global"
  tags              = local.tags
}

resource "ibm_resource_key" "cos_key" {
  name              = "${local.name}-cos-key"
  role                 = "Writer"
  resource_instance_id = ibm_resource_instance.cos.id
  tags              = local.tags

  # TODO private only
  #parameters = {
    # service-endpoints = "private"
    # HMAC              = true
  #}
}

resource "ibm_iam_authorization_policy" "cos_policy" {
  source_service_name         = "cloud-object-storage"
  source_resource_instance_id = ibm_resource_instance.cos.guid
  target_service_name         = data.terraform_remote_state.kms.outputs.key.type
  target_resource_instance_id = data.terraform_remote_state.kms.outputs.kms.guid
  roles                       = ["Reader"]
}
resource "ibm_cos_bucket" "flowlogs" {
  # todo
  #depends_on = [
  #  ibm_iam_authorization_policy.cos_policy
  #]

  bucket_name          = "${local.name}-${local.unique_id}-flowlogs"
  #todo
  #key_protect          = ibm_kms_key.key.crn
  resource_instance_id = ibm_resource_instance.cos.id
  region_location      = var.region
  storage_class        = "smart"

  # set an expiration on the bucket so that flow logs do not accumulate
  expire_rule {
    days   = 1
    enable = true
  }
}

resource "ibm_cos_bucket" "data" {
  # todo
  #depends_on = [
  #  ibm_iam_authorization_policy.cos_policy
  #]

  bucket_name          = "${local.name}-${local.unique_id}-data"
  #todo
  #key_protect          = ibm_kms_key.key.crn
  resource_instance_id = ibm_resource_instance.cos.id
  region_location      = var.region
  storage_class        = "smart"

  dynamic "metrics_monitoring" {
    for_each = var.platform_metrics_crn != "" ? [var.platform_metrics_crn] : []
    content {
      metrics_monitoring_crn = var.platform_metrics_crn
      usage_metrics_enabled  = true
    }
  }

  activity_tracking {
    read_data_events     = true
    write_data_events    = true
    activity_tracker_crn = var.activity_tracker_crn
  }
}

output logdna {
  sensitive = true
  value = {
    ingestion_key = ibm_resource_key.logdna.credentials.ingestion_key
  }
}
output secrets_manager {
  value = {
    instance_id = ibm_resource_instance.secrets_manager.id
  }
}


resource "ibm_resource_instance" "keyprotect" {
  name              = "${var.basename}-kms"
  resource_group_id = local.resource_group
  service           = "kms"
  plan              = "tiered-pricing"
  location          = var.region
  tags              = local.tags
}
resource "ibm_kms_key" "key" {
  instance_id  = ibm_resource_instance.keyprotect.guid
  key_name     = "root_key"
  standard_key = false
  force_delete = true
}
output "key" {
  value     = ibm_kms_key.key
  sensitive = true
}

resource "ibm_iam_authorization_policy" "flowlogs_policy" {
  source_service_name         = "is"
  source_resource_type        = "flow-log-collector"
  target_service_name         = "cloud-object-storage"
  target_resource_instance_id = ibm_resource_instance.cos.guid
  roles                       = ["Writer"]
}

output "flowlogs_bucket_name" {
  value = ibm_cos_bucket.flowlogs.bucket_name
}


--------------------------------------------------------------------------------*/
