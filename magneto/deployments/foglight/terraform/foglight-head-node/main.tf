resource "openstack_compute_keypair_v2" "headnode" {
  name       = "headnode"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

resource "openstack_networking_network_v2" "terraform" {
  name           = "terraform"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "terraform" {
  name            = "terraform"
  network_id      = "${openstack_networking_network_v2.terraform.id}"
  cidr            = "${var.network}"
  ip_version      = 4
  dns_nameservers = "${var.nameservers}"
}

resource "openstack_networking_router_v2" "terraform" {
  name             = "terraform"
  admin_state_up   = "true"
  external_gateway = "${var.external_gateway}"
}

resource "openstack_networking_router_interface_v2" "terraform" {
  router_id = "${openstack_networking_router_v2.terraform.id}"
  subnet_id = "${openstack_networking_subnet_v2.terraform.id}"
}

resource "openstack_compute_secgroup_v2" "headnode" {
  name        = "headnode"
  description = "Security group for the head node"

  rule {
    from_port   = 22
    to_port     = 22
    ip_protocol = "tcp"
    cidr        = "0.0.0.0/0"
  }

  rule {
    from_port   = 80
    to_port     = 80
    ip_protocol = "tcp"
    cidr        = "0.0.0.0/0"
  }

  rule {
    from_port   = -1
    to_port     = -1
    ip_protocol = "icmp"
    cidr        = "0.0.0.0/0"
  }
}

resource "openstack_networking_floatingip_v2" "headnode" {
  pool       = "${var.pool}"
  depends_on = ["openstack_networking_router_interface_v2.terraform"]
}

resource "openstack_compute_instance_v2" "headnode" {
  name            = "headnode"
  image_name      = "${var.image}"
  flavor_name     = "${var.flavor}"
  key_pair        = "${openstack_compute_keypair_v2.headnode.name}"
  security_groups = ["${openstack_compute_secgroup_v2.headnode.name}", "default"]

  network {
    uuid = "${openstack_networking_network_v2.terraform.id}"
  }

  user_data = "#cloud-config\nmanage_etc_hosts: true"
}

resource "openstack_compute_floatingip_associate_v2" "headnode" {
  floating_ip = "${openstack_networking_floatingip_v2.headnode.address}"
  instance_id = "${openstack_compute_instance_v2.headnode.id}"
}

resource "openstack_blockstorage_volume_v2" "scratch" {
  name = "headnode-scratch"
  size = 200
}

resource "openstack_compute_volume_attach_v2" "scratch" {
  instance_id = "${openstack_compute_instance_v2.headnode.id}"
  volume_id = "${openstack_blockstorage_volume_v2.scratch.id}"
}

# Provision the host.
resource "null_resource" "provision" {
  depends_on = ["openstack_compute_floatingip_associate_v2.headnode"]
  connection {
    user     = "${var.ssh_user_name}"
    private_key = "${file(var.ssh_key_file)}"
    host = "${openstack_networking_floatingip_v2.headnode.address}"
  }

  # Running a script or scripts.
  provisioner "remote-exec" {
    scripts = [
      "scripts/nginx.sh",
      "scripts/terraform-prep.sh",
    ]
  }

  /* When a script requires an argument, we have to copy it to the host
   * before we can run it. */
  provisioner "file" {
    source = "scripts/git-shed.sh"
    destination = "/tmp/git-shed.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/git-shed.sh",
      "/tmp/git-shed.sh ${var.github_token}",
    ]
  }

  # This seems less than graceful, but I need those security updates.
  provisioner "remote-exec" {
    script = "scripts/update-ubuntu.sh"
  }
}

# Format and mount the scratch disk.
resource "null_resource" "scratch" {
  depends_on = [
    "openstack_blockstorage_volume_v2.scratch",
    "openstack_compute_volume_attach_v2.scratch",
    "null_resource.provision"
  ]
  connection {
    user     = "${var.ssh_user_name}"
    private_key = "${file(var.ssh_key_file)}"
    host = "${openstack_networking_floatingip_v2.headnode.address}"
  }

  provisioner "remote-exec" {
    script = "scripts/mount-scratch.sh"
  }
}

