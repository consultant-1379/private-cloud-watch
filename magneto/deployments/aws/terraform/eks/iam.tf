################################
#
# IAM roles for EKS
#
################################

# ------------------------------------------
# The "assume role policies" for EKS and EC2
# ------------------------------------------

# Allow the EKS cluster controller to assume the cluster role.

data "aws_iam_policy_document" "eks-assume-role-policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["eks.amazonaws.com"]
    }
  }
}

# Allow EC2 instances to assume the worker role.

data "aws_iam_policy_document" "ec2-assume-role-policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

locals {
  eks_assume_policy = "${data.aws_iam_policy_document.eks-assume-role-policy.json}"
  ec2_assume_policy = "${data.aws_iam_policy_document.ec2-assume-role-policy.json}"
}

# -------------------------------
# IAM role for cluster controller
# -------------------------------

# Create the role for the EKS cluster controller.

resource "aws_iam_role" "cluster" {
  name = "${var.cluster_name}-cluster-role"
  assume_role_policy = "${local.eks_assume_policy}"
}

# Pull data about EKS Cluster and Service policies.

data "aws_iam_policy" "AmazonEKSClusterPolicy" {
  arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

data "aws_iam_policy" "AmazonEKSServicePolicy" {
  arn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
}

# Attach EKS Cluster and Service policies to the EKS cluster role.

resource "aws_iam_role_policy_attachment" "cluster-AmazonEKSClusterPolicy" {
  role       = "${aws_iam_role.cluster.name}"
  policy_arn = "${data.aws_iam_policy.AmazonEKSClusterPolicy.arn}"
}

resource "aws_iam_role_policy_attachment" "cluster-AmazonEKSServicePolicy" {
  role       = "${aws_iam_role.cluster.name}"
  policy_arn = "${data.aws_iam_policy.AmazonEKSServicePolicy.arn}"
}

# -------------------------------
# IAM role for worker nodes
# -------------------------------

# Create the role for the EKS worker nodes.

resource "aws_iam_role" "worker" {
  name = "${var.cluster_name}-worker-role"
  assume_role_policy = "${local.ec2_assume_policy}"
}

# Pull policy data.

data "aws_iam_policy" "AmazonEKSWorkerNodePolicy" {
  arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

data "aws_iam_policy" "AmazonEKS_CNI_Policy" {
  arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}

data "aws_iam_policy" "AmazonEC2ContainerRegistryReadOnly" {
  arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

# Attach policies to the worker role.

resource "aws_iam_role_policy_attachment" "worker-AmazonEKSWorkerNodePolicy" {
  role       = "${aws_iam_role.worker.name}"
  policy_arn = "${data.aws_iam_policy.AmazonEKSWorkerNodePolicy.arn}"
}

resource "aws_iam_role_policy_attachment" "worker-AmazonEKS_CNI_Policy" {
  role       = "${aws_iam_role.worker.name}"
  policy_arn = "${data.aws_iam_policy.AmazonEKS_CNI_Policy.arn}"
}

resource "aws_iam_role_policy_attachment" "worker-AmazonEC2ContainerRegistryReadOnly" {
  role       = "${aws_iam_role.worker.name}"
  policy_arn = "${data.aws_iam_policy.AmazonEC2ContainerRegistryReadOnly.arn}"
}

# -------------------------------------------------
# IAM instance profile for worker autoscaling group
# -------------------------------------------------

# Define an instance profile to use in the autoscaling group.

resource "aws_iam_instance_profile" "worker" {
  name = "${var.cluster_name}-worker"
  role = "${aws_iam_role.worker.name}"
}
