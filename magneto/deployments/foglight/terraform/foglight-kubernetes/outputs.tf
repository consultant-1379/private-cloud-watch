output "kubernetes-master" {
	value = "${openstack_compute_instance_v2.kubernetes_master.access_ip_v4}"
}

output "kubernetes-workers" {
	value = "${join(", ", openstack_compute_instance_v2.kubernetes_worker.*.access_ip_v4)}"
}
