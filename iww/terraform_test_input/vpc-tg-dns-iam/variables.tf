variable ibmcloud_api_key {}
variable resource_group_name {}
variable basename {
  default="iwwts"
}
variable "region" {
  default = "us-south"
}
variable transit_gateway {
  default = true
}
variable shared_lb {
  default = false
}

variable profile {
  default = "cx2-2x4"
}

# note changing image to a different linux (ubuntu to centos for example) requires changing
# user_data for the instances as well.  Changing versions of ubuntu will likely work
variable image {
  # default = "ibm-centos-7-6-minimal-amd64-2"
  default = "ibm-ubuntu-20-04-2-minimal-amd64-1"
}
