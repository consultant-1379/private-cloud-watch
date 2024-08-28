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

variable "network" {
	default = "terraform"
}

variable "managers" {
	default = 3
}

variable "etcd_port" {
	default = 2379
}
