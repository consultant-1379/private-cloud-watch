output "packer-address" {
  value = "${openstack_networking_floatingip_v2.packer.address}"
}
