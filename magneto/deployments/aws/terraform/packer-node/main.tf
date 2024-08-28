# Node template used for packer images in AWS.

variable "name" {}
variable "key" {}
variable "image" { }
variable "subnet" { }
variable "private_dns_zone" {}
variable "availability_zone" { }
variable "create_public_dns_record" { default = false }
variable "public_dns_zone" { default = "" }
variable "private_ips" { default = [] }
variable "security_groups" { default = [] }
variable "volumes" { default = [] }
variable "size" { default = "t3.micro" }

output "id" {
  value = "${aws_instance.node.id}"
}

output "availability_zone" {
  value = "${aws_instance.node.availability_zone}"
}

output "public_ip" {
  value = "${aws_instance.node.public_ip}"
}

# Pull the ami id from aws using a search for the image name.
data "aws_ami" "node" {
  most_recent = true
  owners = ["self"]

  filter {
    name = "name"
    values = ["${var.image}"]
  }
}

# Create the default network interface.
resource "aws_network_interface" "node" {
  subnet_id = "${var.subnet}"
  private_ips = ["${var.private_ips}"]
  security_groups = ["${var.security_groups}"]
}

# Create the instance.
resource "aws_instance" "node" {
  ami = "${data.aws_ami.node.image_id}"
  instance_type = "${var.size}"
  key_name = "${var.key}"
  availability_zone = "${var.availability_zone}"

  # DigitalOcean sets the hostname automatically. Amazon does not, so we do
  # this with userdata instead. This line looks awkward, though. It's
  # terraform's fault.

  user_data = <<EOF
#cloud-config
hostname: ${var.name}
manage_etc_hosts: true
EOF

  network_interface {
    network_interface_id = "${aws_network_interface.node.id}"
    device_index = 0
  }

  tags {
    Name = "${var.name}"
  }
}

# Create a private DNS record for this instance.
resource "aws_route53_record" "private" {
  zone_id = "${var.private_dns_zone}"
  name = "${var.name}"
  type = "A"
  ttl = "300"
  records = ["${var.private_ips}"]
}

# Create a public DNS record for this instance if "create_public_dns_record"
# is set to true. No "if" statements in Terraform, so we pass a boolean to
# "count", which executes once if true.
resource "aws_route53_record" "public" {
  count = "${var.create_public_dns_record}"
  zone_id = "${var.public_dns_zone}"
  name = "${var.name}"
  type = "A"
  ttl = "300"
  records = ["${aws_instance.node.public_ip}"]
}
