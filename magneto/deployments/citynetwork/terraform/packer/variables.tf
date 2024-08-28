variable "image" {
  default = "Ubuntu 16.04 Xenial Xerus"
}

variable "flavor" {
  default = "1C-1GB"
}

variable "ssh_key_file" {
  default = "~/.ssh/id_rsa.terraform"
}

variable "ssh_user_name" {
  default = "ubuntu"
}

# Note: the external network id is different for any given region. To get this
# id, you have to run "openstack network list --external" and pick an external
# network. I'm not sure if there is a better way to do this that would work for
# all regions, but I can't find one yet. I'm also not sure if this value ever
# changes...

variable "external_network_id" {
  default = "fba95253-5543-4078-b793-e2de58c31378"
}

# This name also comes from "openstack network list --external" output.

variable "pool" {
  default = "ext-net"
}

variable "subnet" {
  default = "10.255.255.0/24"
}

variable "region" {
  default = "Kna1"
}
