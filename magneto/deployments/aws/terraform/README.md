## What's this?

This is the terraform configuration that is used to create all first-level
resources in AWS, starting with the jump node. A head node hosting a docker
registry will be added soon, along with a DNS subdomain delegated to Route53.

At the moment, these nodes will use the Erix central LDAP and Vault
infrastructure in DigitalOcean.

## Design goals, philosophy, etc.

Instances are modified using the currently-fashionable "immutable" method (i.e.
in order to make a change, we rebuild everything). Each node is built using a
"generic-node" module which is provided here, and then customized using a
"null_resource" provisioner.

Instances are assumed to be ephemeral. The only persistent disk storage is /home
on the jump node; NFS is not used, so if you write something to your home
directory on another node, it will not persist across rebuilds.

## How to run this

You should not run this code unless you know what you are doing. Running this
will destroy and replace infrastructure. If you intend to do that, please
follow along.

If terraform has already been run and resources are built, you'll need the
private ssh key in order to run terraform again. It is currently stored in
Vault at /terraform. If Vault is down, ask around and get it from the person
who ran this last.

In order to run this, the following is expected:

- You should use at least terraform v0.11.1. Earlier versions aren't tested.
- Your working path must be this directory when you run terraform.
- Your AWS credentials must be set. (See the Packer readme for this.)
- You must run `terraform init` first.
- You must run `terraform get` in order to load the "generic-node" module
  before running anything else.

Good luck.
