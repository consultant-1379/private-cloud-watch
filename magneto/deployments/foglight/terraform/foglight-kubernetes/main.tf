# Keypair to copy to each node in the swarm.
resource "openstack_compute_keypair_v2" "kubernetes" {
	name = "kubernetes"
	public_key = "${file("${var.ssh_key_file}.pub")}"
}

# Kubernetes master.
resource "openstack_compute_instance_v2" "kubernetes_master" {
	name = "kubernetes-master"
	image_name = "${var.image}"
	flavor_name = "${var.flavor}"
	key_pair = "${openstack_compute_keypair_v2.kubernetes.name}"
	security_groups = ["default"]

	network {
		name = "${var.network}"
	}

	user_data = "#cloud-config\nmanage_etc_hosts: true"

	connection {
		user = "${var.ssh_user_name}"
		private_key = "${file(var.ssh_key_file)}"
	}

	# Install kubernetes.
	provisioner "remote-exec" {
		script = "scripts/kubernetes-install.sh"
	}

	# Start the kubernetes master using kubeadm.
	provisioner "file" {
		source = "scripts/kubeadm-master.sh"
		destination = "./kubeadm-master.sh"
	}
	provisioner "remote-exec" {
		inline = [
			"chmod +x ./kubeadm-master.sh",
			"./kubeadm-master.sh ${var.api_port} ${var.join_token}",
		]
	}
}

# Kubernetes workers.
resource "openstack_compute_instance_v2" "kubernetes_worker" {
	depends_on = ["openstack_compute_instance_v2.kubernetes_master"]

	count = "${var.workers}"
	name = "kubernetes-worker-${count.index}"
	image_name = "${var.image}"
	flavor_name = "${var.flavor}"
	key_pair = "${openstack_compute_keypair_v2.kubernetes.name}"
	security_groups = ["default"]

	network {
		name = "${var.network}"
	}

	user_data = "#cloud-config\nmanage_etc_hosts: true"

	connection {
		user = "${var.ssh_user_name}"
		private_key = "${file(var.ssh_key_file)}"
	}

	# Install kubernetes.
	provisioner "remote-exec" {
		script = "scripts/kubernetes-install.sh"
	}

	# Join the kubernetes cluster using kubeadm.
	provisioner "file" {
		source = "scripts/kubeadm-worker.sh"
		destination = "./kubeadm-worker.sh"
	}
	provisioner "remote-exec" {
		inline = [
			"chmod +x ./kubeadm-worker.sh",
			"./kubeadm-worker.sh ${openstack_compute_instance_v2.kubernetes_master.access_ip_v4} ${var.api_port} ${var.join_token}",
		]
	}
}
