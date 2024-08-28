variable "image" {
  default = "Ubuntu-17.04-zesty"
}

variable "flavor" {
  default = "c2m4"
}

variable "ssh_key_file" {
  default = "~/.ssh/id_rsa.terraform"
}

variable "ssh_user_name" {
  default = "ubuntu"
}

variable "external_gateway" {
  default = "d6cc381d-38cb-418f-bea1-082a0d5169ee"
}

variable "pool" {
  default = "ECN"
}

variable "network" {
  default = "10.0.0.0/24"
}

variable "nameservers" {
  default = ["136.225.128.194", "136.225.128.195"]
}

variable "github_token" { }
