output "address" {
  value = "${digitalocean_droplet.headnode.ipv4_address}"
}
