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

variable "api_port" {
	default = "6443"
}

variable "join_token" {
	default = "123456.1234567890123456"
}

variable "workers" {
	default = 3
}
