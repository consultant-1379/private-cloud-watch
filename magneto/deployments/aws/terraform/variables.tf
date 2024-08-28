variable "digitalocean_token" {}
variable "github_token" {}
variable "spaces_access_key" {}
variable "spaces_secret_key" {}

variable "spaces_endpoint" {
  default = "nyc3.digitaloceanspaces.com"
}

variable "ssh_key_file" {
  default = "~/.ssh/id_rsa.terraform"
}

variable "domain" {
  default = "erixzone.net"
}

variable "region" {
  default = "us-east-1"
}
