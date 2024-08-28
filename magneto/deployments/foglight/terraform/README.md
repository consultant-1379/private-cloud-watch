Terraform configurations go in this directory.

Configuration descriptions
--------------------------

- foglight-head-node: This is used in order to stand up the head node and
related infrastructure in Foglight (the Ericsson Research OpenStack).
- foglight-docker-swarm: This creates a private docker swarm cluster inside
Foglight. (It must be run from the Foglight head node in order to provision
the nodes properly.)

How to run Terraform against Foglight
-------------------------------------

1. Before you do anything, you'll need to source the necessary rc script and
type in your username and password. This will set up your environment to make
connections to OpenStack. For foglight, this command is: `source
scripts/foglight-erix-openrc.sh`
2. (Optional) You can also `pip install python-openstackclient`, which gives
you the ability to run OpenStack commands at the CLI, but that isn't strictly
needed for Terraform. Furthermore, `pip install python-swiftclient` installs
the client for the Swift object store.
3. To get terraform, you can run `scripts/terraform-prep.sh`, or manually
download terraform and extract it to `/usr/local/bin` instead.
4. If you haven't run Terraform yet in the directory you're sitting in, run
`terraform init <project dir>` to download the necessary providers.
5. You will also need an ssh key at `~/.ssh/id_rsa.terraform` which will be
used to log in to the instances you create. It might already be there; the
`terraform-prep.sh` script creates a new one for you. If someone else has
already created the head node, you'll have to get the ssh key from them.
6. You can run `terraform plan <project dir>` to figure out what terraform
would actually do if you ran `apply`, and then run `terraform apply <dir>` to
apply it. `terraform destroy` will wipe out everything that terraform sees in
its state file.

More Terraform information
--------------------------

Terraform is sensitive about having the proper "state file" around, which it
needs in order to do the right things. By default, it tries to store this
locally, which is not fantastic for team use. I have attempted to smooth this
over by configuring a foglight swift container as a backend. If we are starting
from nuked-scratch, here's the command I used to create that container:

`~/.local/bin/openstack container create terraform-headnode-state`

If you want to experiment without breaking what's already there, you can make a
copy of one of these configurations and point it at a fresh state file, to make
sure you don't clash with anything else that's already running.

Make sure the names of the resources you create are unique. You'll get an error
if you try to create certain types of openstack entities that have the same
names as others (e.g. the openstack keypair).
