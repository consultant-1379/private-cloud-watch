This is a terraform module for a node which is built using a packer baked
image. Without it, a lot of terraform code would have to be retyped every time
you create a node.

You can invoke it like this from your main terraform config, using the image
name that was saved by your packer run:

    module "head_node" {
      source = "./packer-node"
      name = "head"
      image = "head-2018-04-08-1531"
      domain = "${var.domain}"
      region = "${var.region}"
      ssh_key = "${local.ssh_key}"
    }

Note: when you use modules, including ones that are in your local directory,
terraform forces you to run "terraform get" before running anything else, in
order to "load" the module. Pretty janky.
