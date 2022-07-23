output "zone" {
  value = var.zone
}
output "name" {
  value = var.name
}
output "instances" {
  value = ibm_is_instance.zone
}
output "floating_ips" {
  value = ibm_is_floating_ip.zone
}
output "subnet" {
  value = ibm_is_subnet.zone
}
output "subnet_nlb" {
  value = ibm_is_subnet.zone_nlb
}
output "subnet_dns" {
  value = ibm_is_subnet.zone_dns
}