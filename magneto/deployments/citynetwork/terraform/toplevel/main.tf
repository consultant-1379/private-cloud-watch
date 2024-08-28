provider "openstack" {
}

# SSH keypair.
resource "openstack_compute_keypair_v2" "toplevel" {
  name = "toplevel"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

####################################
# Create network, subnet, and router.
####################################

resource "openstack_networking_network_v2" "toplevel" {
  name = "toplevel"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "toplevel" {
  name = "toplevel"
  network_id = "${openstack_networking_network_v2.toplevel.id}"
  cidr = "${var.subnet}"
  ip_version = 4
}

resource "openstack_networking_router_v2" "toplevel" {
  name = "toplevel"
  admin_state_up = "true"
  external_network_id = "${var.external_network_id}"
}

resource "openstack_networking_router_interface_v2" "toplevel" {
  router_id = "${openstack_networking_router_v2.toplevel.id}"
  subnet_id = "${openstack_networking_subnet_v2.toplevel.id}"
}

####################################
# Create jump node.
####################################

resource "openstack_compute_instance_v2" "jump" {
  name            = "jump"
  image_name      = "jump-2018-05-30-1317"
  flavor_name     = "${var.flavor}"
  key_pair        = "${openstack_compute_keypair_v2.toplevel.name}"

  network {
    uuid = "${openstack_networking_network_v2.toplevel.id}"
  }
}

####################################
# Create /home volume for jump node.
####################################

resource "openstack_blockstorage_volume_v2" "jump" {
  name = "jump-home"
  size = 100
}

resource "openstack_compute_volume_attach_v2" "jump" {
  instance_id = "${openstack_compute_instance_v2.jump.id}"
  volume_id = "${openstack_blockstorage_volume_v2.jump.id}"
}

####################################
# Create floating IP for jump node.
####################################

resource "openstack_networking_floatingip_v2" "jump" {
  pool       = "${var.pool}"
  depends_on = ["openstack_networking_router_interface_v2.toplevel"]
}

resource "openstack_compute_floatingip_associate_v2" "jump" {
  floating_ip = "${openstack_networking_floatingip_v2.jump.address}"
  instance_id = "${openstack_compute_instance_v2.jump.id}"
}
