provider "digitalocean" {
  token = "${var.digitalocean_token}"
}

resource "digitalocean_ssh_key" "headnode" {
  name = "Terraform headnode"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

resource "digitalocean_droplet" "headnode" {
  name = "headnode"
  image = "${var.image}"
  region = "${var.region}"
  size = "${var.size}"
  private_networking = true
  ssh_keys = [
    "${digitalocean_ssh_key.headnode.id}"
  ]

  connection {
    user = "root"
    type = "ssh"
    private_key = "${file(var.ssh_key_file)}"
  }

  provisioner "remote-exec" {
    scripts = [
      "../scripts/nginx.sh",
      "../scripts/terraform-prep.sh",
    ]
  }
}
