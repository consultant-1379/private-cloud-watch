# Keypair to copy to each node in the swarm.
resource "openstack_compute_keypair_v2" "swarm" {
	name = "docker-swarm"
	public_key = "${file("${var.ssh_key_file}.pub")}"
}

# This etcd scratchpad is used by swarmer to bring up the swarm.
resource "openstack_compute_instance_v2" "etcd" {
	name = "etcd-swarmer"
	image_name = "${var.image}"
	flavor_name = "${var.flavor}"
	key_pair = "${openstack_compute_keypair_v2.swarm.name}"
	security_groups = ["default"]

	network {
		name = "${var.network}"
	}

	user_data = "#cloud-config\nmanage_etc_hosts: true"

	connection {
		user = "${var.ssh_user_name}"
		private_key = "${file(var.ssh_key_file)}"
	}

	# Start the etcd scratchpad.
	provisioner "file" {
		source = "scripts/etcd-swarmer.sh"
		destination = "./etcd-swarmer.sh"
	}

	provisioner "remote-exec" {
		inline = [
			"chmod +x ./etcd-swarmer.sh",
			"./etcd-swarmer.sh ${self.access_ip_v4} ${var.etcd_port}",
		]
	}
}

# Each swarm node is provisioned identically and pointed at Etcd.
resource "openstack_compute_instance_v2" "swarm_manager" {
	depends_on = ["openstack_compute_instance_v2.etcd"]

	count = "${var.managers}"
	name = "swarm-manager-${count.index}"
	image_name = "${var.image}"
	flavor_name = "${var.flavor}"
	key_pair = "${openstack_compute_keypair_v2.swarm.name}"
	security_groups = ["default"]

	network {
		name = "${var.network}"
	}

	user_data = "#cloud-config\nmanage_etc_hosts: true"

	connection {
		user = "${var.ssh_user_name}"
		private_key = "${file(var.ssh_key_file)}"
	}

	# Install docker.
	provisioner "remote-exec" {
		script = "scripts/docker.sh"
	}

	# Copy swarmer binary to the scripts dir.
	provisioner "file" {
		source = "/home/ubuntu/shed/swarmer/swarmer"
		destination = "./swarmer"
	}

	# Also the swarmer start script.
	provisioner "file" {
		source = "scripts/swarmer.sh"
		destination = "./swarmer.sh"
	}

	# Start up the swarmer.
	provisioner "remote-exec" {
		inline = [
			"chmod +x ./swarmer.sh ./swarmer",
			"./swarmer.sh ${openstack_compute_instance_v2.etcd.access_ip_v4} ${var.etcd_port} manager --debug",
		]
	}
}
