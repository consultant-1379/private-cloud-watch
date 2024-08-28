variable "region" {
  default = "us-east-1"
}

variable "replicas" {
  default = 12
}

variable "beacon_port" {
  default = 29718
}

variable "flock_port" {
  default = 23123
}

variable "flock_key" {
  default = "27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf"
}

variable "cidr_block" {
  default = "10.0.0.0/22"
}

variable "github_token" {}
