################################
#
# EKS worker autoscaling group
#
################################

# Fetch information about the EKS worker AMI.

data "aws_ami" "worker" {
  filter {
    name   = "name"
    values = ["${var.ami_worker_name}"]
  }
  # This is the Amazon EKS AMI account ID.
  owners      = ["${var.ami_account_id}"]
  most_recent = true
}

# This userdata script runs the EKS bootstrap script which is provided in the
# AMI. It does not, however, fully enable the worker to join the cluster.
# There is a ConfigMap that needs to be applied at the command line for that to
# happen. Check the README for instructions on how to do this.
#
# For even more info, read this:
# https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html

locals {
  worker-userdata = <<EOF
#!/bin/bash

set -o xtrace

/etc/eks/bootstrap.sh \
    --apiserver-endpoint '${aws_eks_cluster.master.endpoint}' \
    --b64-cluster-ca '${aws_eks_cluster.master.certificate_authority.0.data}' \
    '${var.cluster_name}'

EOF
}

# Create a launch configuration for the workers.

resource "aws_launch_configuration" "worker" {
  name_prefix = "terraform-eks-${var.cluster_name}-launch"
  image_id = "${data.aws_ami.worker.id}"
  instance_type = "${var.worker_type}"
  associate_public_ip_address = true
  key_name = "${var.ssh_key_name}"
  iam_instance_profile = "${aws_iam_instance_profile.worker.name}"
  security_groups = ["${aws_security_group.worker.id}"]
  user_data_base64 = "${base64encode(local.worker-userdata)}"

  lifecycle {
    create_before_destroy = true
  }
}

# Create an autoscaling group for the workers.

resource "aws_autoscaling_group" "worker" {
  name = "terraform-eks-${var.cluster_name}-worker"
  launch_configuration = "${aws_launch_configuration.worker.id}"
  min_size = 1
  desired_capacity = "${var.worker_count}"
  max_size = "${var.worker_count * var.autoscaling_multiplier}"
  vpc_zone_identifier = ["${aws_subnet.public.*.id}"]

  tag {
    key = "Name"
    value = "terraform-eks-${var.cluster_name}-worker"
    propagate_at_launch = true
  }

  tag {
    key = "kubernetes.io/cluster/${var.cluster_name}"
    value = "owned"
    propagate_at_launch = true
  }
}
