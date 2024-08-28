output "etcd-swarm" {
	value = "${openstack_compute_instance_v2.etcd.access_ip_v4}"
}

output "swarm-managers" {
	value = "${join(", ", openstack_compute_instance_v2.swarm_manager.*.access_ip_v4)}"
}
