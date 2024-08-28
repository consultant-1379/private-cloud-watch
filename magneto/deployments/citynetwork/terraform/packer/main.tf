provider "openstack" {
}

# SSH keypair.
resource "openstack_compute_keypair_v2" "packer" {
  name = "packer"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

####################################
# Create network, subnet, and router.
####################################

resource "openstack_networking_network_v2" "packer" {
  name = "packer"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "packer" {
  name = "packer"
  network_id = "${openstack_networking_network_v2.packer.id}"
  cidr = "${var.subnet}"
  ip_version = 4
}

resource "openstack_networking_router_v2" "packer" {
  name = "packer"
  admin_state_up = "true"
  external_network_id = "${var.external_network_id}"
}

resource "openstack_networking_router_interface_v2" "packer" {
  router_id = "${openstack_networking_router_v2.packer.id}"
  subnet_id = "${openstack_networking_subnet_v2.packer.id}"
}

####################################
# Create packer node.
####################################

resource "openstack_compute_instance_v2" "packer" {
  name            = "packer"
  image_name      = "${var.image}"
  flavor_name     = "${var.flavor}"
  key_pair        = "${openstack_compute_keypair_v2.packer.name}"

  network {
    uuid = "${openstack_networking_network_v2.packer.id}"
  }
}

####################################
# Create floating IP.
####################################

resource "openstack_networking_floatingip_v2" "packer" {
  pool       = "${var.pool}"
  depends_on = ["openstack_networking_router_interface_v2.packer"]
}

resource "openstack_compute_floatingip_associate_v2" "packer" {
  floating_ip = "${openstack_networking_floatingip_v2.packer.address}"
  instance_id = "${openstack_compute_instance_v2.packer.id}"
}

####################################
# Provision using the floating IP.
####################################

resource "null_resource" "provision" {
  depends_on = ["openstack_compute_floatingip_associate_v2.packer"]

  connection {
    user     = "${var.ssh_user_name}"
    private_key = "${file(var.ssh_key_file)}"
    host = "${openstack_networking_floatingip_v2.packer.address}"
  }

  provisioner "remote-exec" {
    script = "scripts/provision-packer-node.sh"
  }
}
