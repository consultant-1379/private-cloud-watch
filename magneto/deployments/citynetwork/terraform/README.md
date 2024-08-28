Terraform configs here:

packer - Creates a temporary network and node for running Packer. This is
needed because Packer has to be run from inside the City Network environment
due to how the default networking works there.

toplevel - Creates basic networking and a jump node. Other top level services
will go here eventually.

