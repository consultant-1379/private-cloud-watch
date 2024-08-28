variable "ssh_key_name" {
  default = "terraform-eks"
}

variable "ssh_key_file" {
  default = "~/.ssh/id_rsa.terraform"
}

variable "region" {
  default = "us-east-1"
}

variable "cluster_name" {
  default = "erix"
}

variable "cidr_block" {
  default = "10.0.0.0/22"
}

variable "subnet_bits" {
  default = "2"
}

# How many availability zones should we put workers in?
variable "availability_zones" {
  default = "3"
}

# Workers have 2 GB of RAM by default...due to cheepnis.
variable "worker_type" {
  default = "t3.small"
}

# How many workers desired?
variable "worker_count" {
  default = "3"
}

# Autoscaling group will allow a max of (multiplier x worker_count).
variable "autoscaling_multiplier" {
  default = "2"
}

variable "ami_account_id" {
  default = "602401143452"
}

variable "ami_worker_name" {
  default = "amazon-eks-node-1.11-v20190329"
}
