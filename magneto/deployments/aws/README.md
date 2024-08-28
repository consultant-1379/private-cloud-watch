These files create the resources in AWS that are intended to be part of an ENM
cloud PoC as of September 2018.

It uses the same three-phase deployment mechanism as the erix deployment in
digitalocean:

1. Packer uses Ansible to build an image
2. Packer saves the image as a DigitalOcean snapshot
3. Terraform deploys the image to DigitalOcean


