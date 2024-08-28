################################
#
# EKS networking
#
################################

# ------------------------------
# VPC for EKS
# ------------------------------

# Create a VPC for EKS.

resource "aws_vpc" "eks" {
  cidr_block = "${var.cidr_block}"

  enable_dns_support = true
  enable_dns_hostnames = true

  # As far as I can tell, these kubernetes.io tags are necessary.
  # The AWS docs aren't clear about it, though, so I don't know.
  tags = "${map(
    "Name", "${var.cluster_name}-vpc",
    "kubernetes.io/cluster/${var.cluster_name}", "shared"
  )}"
}

locals {
  vpc = "${aws_vpc.eks.id}"
}

# ------------------------------
# Subnets for EKS
# ------------------------------

# The cloudformation template in the EKS getting started guide includes three
# subnets. EKS requires at least two, from different availability zones. This
# is set up to use however many are set in the "availability_zones" variable.
# We load the availability zones in using a data object so we don't have to
# type them.

data "aws_availability_zones" "az" {}

locals {
  availability_zones = "${slice(data.aws_availability_zones.az.names, 0, var.availability_zones)}"
}

# Create one subnet per availability zone.

resource "aws_subnet" "public" {
  count = "${length(local.availability_zones)}"
  vpc_id = "${local.vpc}"
  cidr_block = "${cidrsubnet(aws_vpc.eks.cidr_block, var.subnet_bits, count.index + 1)}"
  availability_zone = "${local.availability_zones[count.index]}"

  tags = "${map(
    "Name", "${var.cluster_name}-subnet-public-${local.availability_zones[count.index]}",
    "kubernetes.io/cluster/${var.cluster_name}", "shared"
  )}"
}

# ------------------------------
# Routing configuration
# ------------------------------

# This gateway routes packets from the public subnet to the internet.

resource "aws_internet_gateway" "eks" {
  vpc_id = "${local.vpc}"

  tags = "${map(
    "Name", "${var.cluster_name}-igw",
    "kubernetes.io/cluster/${var.cluster_name}", "shared"
  )}"
}

# This route table will be used as the main route table for the VPC.

resource "aws_route_table" "main" {
  vpc_id = "${local.vpc}"

  tags = "${map(
    "Name", "${var.cluster_name}-routetable"
  )}"
}

resource "aws_main_route_table_association" "main" {
  vpc_id = "${local.vpc}"
  route_table_id = "${aws_route_table.main.id}"
}

# We add a default route to the internet gateway.

resource "aws_route" "default" {
  destination_cidr_block = "0.0.0.0/0"
  route_table_id = "${aws_route_table.main.id}"
  gateway_id = "${aws_internet_gateway.eks.id}"
}

