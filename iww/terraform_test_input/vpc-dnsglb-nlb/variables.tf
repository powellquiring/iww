# variables - see template.local.env for the required variables

variable "prefix" {
  description = "resources created will be named: $${prefix}vpc-pubpriv, vpc name will be $${prefix} or will be defined by vpc_name"
  default     = "iww-glbnlb"
}

variable "resource_group_name" {
  description = "Resource group that will contain all the resources created by the script."
}

variable "ssh_key_name" {
  description = "SSH keys are needed to connect to virtual instances. https://cloud.ibm.com/docs/vpc?topic=vpc-getting-started"
}

variable "region" {
  description = "Availability zone that will have the resources deployed to.  To obtain a list of availability zones you can run the ibmcloud cli: ibmcloud is regions."
  default     = "us-south"
}

variable "zones" {
  description = "number of zones to create"
  default     = 2
}

variable "instances" {
  description = "number of instances per zone connected to the NLB in the zone"
  default     = 1
}