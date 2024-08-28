################################
#
# Security groups for EKS
#
################################

# --------------------------
# Security group for cluster
# --------------------------

resource "aws_security_group" "cluster" {
  name = "${var.cluster_name}-sg-cluster"
  description = "Cluster controller security group"
  vpc_id = "${local.vpc}"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "${var.cluster_name}-sg-cluster"
  }
}

# Allow communication from jump node to cluster API.

# Note: there's no good way to make a resource depend on whether data has been
# discovered, so if the jump node doesn't exist when this is run, the security
# group rule afterwards will bomb out. You can uncomment it if you need it.

#data "aws_instance" "jump" {
#  filter {
#    name = "tag:Name"
#    values = ["jump"]
#  }
#}
#
#resource "aws_security_group_rule" "jump-to-cluster" {
#  cidr_blocks = ["${data.aws_instance.jump.public_ip}/32"]
#  description = "Allow jump to communicate with cluster API"
#  type = "ingress"
#  protocol = "tcp"
#  from_port = 443
#  to_port = 443
#  security_group_id = "${aws_security_group.cluster.id}"
#}

# --------------------------
# Security group for workers
# --------------------------

resource "aws_security_group" "worker" {
  name = "${var.cluster_name}-sg-worker"
  description = "Worker nodes security group"
  vpc_id = "${local.vpc}"

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "${var.cluster_name}-sg-worker"
  }
}

# Allow communication among workers on any port.

resource "aws_security_group_rule" "worker-to-worker" {
  description = "Allow worker communication"
  type = "ingress"
  protocol = "-1"
  from_port = 0
  to_port = 65535
  security_group_id = "${aws_security_group.worker.id}"
  source_security_group_id = "${aws_security_group.worker.id}"
}

# Allow communication from cluster controller to TCP ports above 1024.

resource "aws_security_group_rule" "cluster-to-worker" {
  description = "Allow cluster controller to communicate on ports above 1024"
  type = "ingress"
  protocol = "tcp"
  from_port = 1025
  to_port = 65535
  security_group_id = "${aws_security_group.worker.id}"
  source_security_group_id = "${aws_security_group.cluster.id}"
}

# Allow communication from workers to cluster controller on 443.

resource "aws_security_group_rule" "worker-to-cluster" {
  description = "Allow workers to contact cluster controller"
  type = "ingress"
  protocol = "tcp"
  from_port = 443
  to_port = 443
  security_group_id = "${aws_security_group.cluster.id}"
  source_security_group_id = "${aws_security_group.worker.id}"
}

# Allow SSH from jump node to workers.

#resource "aws_security_group_rule" "jump-to-worker" {
#  cidr_blocks = ["${data.aws_instance.jump.public_ip}/32"]
#  description = "Allow jump to communicate with cluster API"
#  type = "ingress"
#  protocol = "tcp"
#  from_port = 22
#  to_port = 22
#  security_group_id = "${aws_security_group.worker.id}"
#}
