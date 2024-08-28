################################
#
# Global
#
################################

provider "aws" {
  region = "${var.region}"
}

# We delegate our subdomain in the erixzone.net domain at DigitalOcean.
provider "digitalocean" {
  token = "${var.digitalocean_token}"
}

# Global root ssh key.
resource "aws_key_pair" "global" {
  key_name = "terraform-global"
  public_key = "${file("${var.ssh_key_file}.pub")}"
}

# Just use the first availability zone.
data "aws_availability_zones" "az" {}

locals {
  key = "${aws_key_pair.global.key_name}"
  availability_zone = "${data.aws_availability_zones.az.names[0]}"
}


################################
#
# Jump and head nodes
#
################################

# ------------------------------
# VPC and subnets for our hosts
# ------------------------------

# We create a VPC for all of the terraform hosts.

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"

  # Enable local DNS support, so we can get to other hosts in the VPC by name.
  enable_dns_support = true
  enable_dns_hostnames = true

  tags {
    Name = "terraform-main"
  }
}

locals {
  vpc = "${aws_vpc.main.id}"
}

# The public subnet gets connectivity to the internet and a public IP.

resource "aws_subnet" "public" {
  vpc_id = "${local.vpc}"
  cidr_block = "10.0.1.0/24"
  availability_zone = "${local.availability_zone}"
  map_public_ip_on_launch = true

  tags {
    Name = "terraform-public"
  }
}

# The private subnet can only contact the other subnets in our VPC. Currently,
# it has no connectivity to the internet, although this can be set up using a
# NAT gateway.

resource "aws_subnet" "private" {
  vpc_id = "${local.vpc}"
  cidr_block = "10.0.2.0/24"
  availability_zone = "${local.availability_zone}"

  tags {
    Name = "terraform-private"
  }
}

locals {
  public_subnet = "${aws_subnet.public.id}"
  private_subnet = "${aws_subnet.private.id}"
}

# ------------------------------
# Routing configuration
# ------------------------------

# This gateway routes packets from the public subnet to the internet.

resource "aws_internet_gateway" "public" {
  vpc_id = "${local.vpc}"

  tags {
    Name = "terraform-public"
  }
}

# This route table gets associated with the public subnet so that the public
# hosts send their packets to the internet via the internet gateway.

resource "aws_route_table" "public" {
  vpc_id = "${local.vpc}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.public.id}"
  }

  tags {
    Name = "terraform-public"
  }
}

resource "aws_route_table_association" "public" {
  subnet_id = "${local.public_subnet}"
  route_table_id = "${aws_route_table.public.id}"
}

# You also need a NAT gateway to be defined here if you want instances in the
# private space to be able to contact the internet.

# ------------------------------
# Security groups
# ------------------------------

resource "aws_security_group" "allow_ssh" {
  name = "allow_ssh"
  description = "Allow inbound ssh connections"
  vpc_id = "${local.vpc}"

  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_ssh"
  }
}

resource "aws_security_group" "allow_https" {
  name = "allow_https"
  description = "Allow inbound https connections"
  vpc_id = "${local.vpc}"

  ingress {
    from_port = 443
    to_port = 443
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_https"
  }
}

resource "aws_security_group" "allow_ping" {
  name = "allow_ping"
  description = "Allow inbound pings"
  vpc_id = "${local.vpc}"

  ingress {
    from_port = 8
    to_port = 0
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_ping"
  }
}

resource "aws_security_group" "allow_mtu_path" {
  name = "allow_mtu_path"
  description = "Allow mtu path discovery"
  vpc_id = "${local.vpc}"

  ingress {
    from_port = 3
    to_port = 4
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_mtu_path"
  }
}

resource "aws_security_group" "allow_outbound" {
  name = "allow_outbound"
  description = "Allow all outbound connections"
  vpc_id = "${local.vpc}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_outbound"
  }
}

locals {
  allow_ssh = "${aws_security_group.allow_ssh.id}"
  allow_https = "${aws_security_group.allow_https.id}"
  allow_ping = "${aws_security_group.allow_ping.id}"
  allow_mtu_path = "${aws_security_group.allow_mtu_path.id}"
  allow_outbound = "${aws_security_group.allow_outbound.id}"
}

# ------------------------------
# DNS zones
# ------------------------------

# Private domain for the VPC.
resource "aws_route53_zone" "private" {
  name = "erix.vpc"
  vpc {
    vpc_id = "${local.vpc}"
  }

  tags {
    Name = "terraform-private"
  }
}

# Public domain for erixzone.net, delegated from DigitalOcean.
resource "aws_route53_zone" "public" {
  name = "aws.erixzone.net"

  tags {
    Name = "terraform-public"
  }
}

locals {
  private_dns_zone = "${aws_route53_zone.private.zone_id}"
  public_dns_zone = "${aws_route53_zone.public.zone_id}"
}

# Add NS records for our subdomain at DigitalOcean.
resource "digitalocean_record" "aws-ns" {
  count = 4
  domain = "erixzone.net"
  type = "NS"
  name = "aws"
  value = "${aws_route53_zone.public.name_servers[count.index]}."
}

# ------------------------------
# Head node
# ------------------------------

module "head_node" {
  source = "./packer-node"
  name = "head"
  image = "head-2018-10-17-1459"
  key = "${local.key}"
  subnet = "${local.public_subnet}"
  private_ips = ["10.0.1.10"]
  private_dns_zone = "${local.private_dns_zone}"
  create_public_dns_record = true
  public_dns_zone = "${local.public_dns_zone}"
  availability_zone = "${local.availability_zone}"
  security_groups = [
    "${local.allow_ssh}",
    "${local.allow_https}",
    "${local.allow_ping}",
    "${local.allow_mtu_path}",
    "${local.allow_outbound}"
  ]
}

resource "aws_route53_record" "registry" {
  zone_id = "${local.public_dns_zone}"
  name = "registry"
  type = "A"
  ttl = "300"
  records = ["${module.head_node.public_ip}"]
}

# ------------------------------
# Jump node
# ------------------------------

module "jump_node" {
  source = "./packer-node"
  name = "jump"
  image = "jump-2018-10-17-1453"
  key = "${local.key}"
  subnet = "${local.public_subnet}"
  private_ips = ["10.0.1.11"]
  private_dns_zone = "${local.private_dns_zone}"
  create_public_dns_record = true
  public_dns_zone = "${local.public_dns_zone}"
  availability_zone = "${local.availability_zone}"
  security_groups = [
    "${local.allow_ssh}",
    "${local.allow_ping}",
    "${local.allow_mtu_path}",
    "${local.allow_outbound}"
  ]
}

# Persistent /home volume for jump node.
resource "aws_ebs_volume" "jump_home" {
  availability_zone = "${local.availability_zone}"
  size = 50
  tags {
    Name = "jump-home"
  }
}

# Attachment for /home volume.
resource "aws_volume_attachment" "jump_home" {
  device_name = "/dev/sde"
  volume_id = "${aws_ebs_volume.jump_home.id}"
  instance_id = "${module.jump_node.id}"
}

