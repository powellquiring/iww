variable "ibmcloud_api_key" {}
variable "resource_group_name" {}

variable "basename" {
  default="iwwvpc3"
}
variable "ssh_key_name" {
  default = "pfq"
}
variable "region" {
  default = "us-south"
}

# set to true to enable logdna and sysdig
variable "observability" {
  default = true
}

variable "subnets" {
  default = 2
}
variable "profile" {
  default = "cx2-2x4"
}
variable "image_name" {
  default = "ibm-ubuntu-20-04-minimal-amd64-2"
}
variable "postgresql" {
  type = bool
  default = false
}