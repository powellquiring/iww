variable "ibmcloud_api_key" {}
variable "resource_group_name" {}
variable "basename" {
  default = "iwwts"
}
variable "region" {
  default = "us-south"
}
variable "subnets" {
  default = 2
}
variable "profile" {
  default = "cx2d-2x4"
}
variable "image_name" {
  default = "ibm-ubuntu-20-04-minimal-amd64-2"
}

variable "ssh_key_name" {
  default = "pfq"
}
