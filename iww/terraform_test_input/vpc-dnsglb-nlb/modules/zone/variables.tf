variable "name" {}
variable "zone" {}
variable "cidr" {}
variable "profile" {}
variable "image" {}
variable "user_data" {}
variable "instances" {} # number of ibm_is_instance to create

# resources ibm_is_*
variable "vpc" {}
variable "resource_group" {}
variable "keys" {} # list of ibm_is_key
