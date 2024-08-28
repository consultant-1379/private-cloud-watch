################################
#
# Configuration
#
################################

provider "aws" {
  region = "${var.region}"
}

# Create root ssh key to use for EKS workers.
resource "aws_key_pair" "eks" {
  key_name = "${var.ssh_key_name}"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}
