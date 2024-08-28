variable "digitalocean_token" {}
variable "github_token" {}

variable "spaces_endpoint" {
  default = "https://nyc3.digitaloceanspaces.com/"
}

variable "ssh_key_file" {
  default = "~/.ssh/id_rsa.terraform"
}

variable "image" {
  default = "ubuntu-16-04-x64"
}

variable "region" {
  default = "nyc3"
}

variable "size" {
  default = "2gb"
}
