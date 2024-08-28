Here are some scripts which, together, attempt to perform the steps necessary
to install ENM in openstack.

They're meant to be run in this order:

1. ./download-media.sh enm-media-2017-10.txt downloads/
2. ./prepare-artifacts.sh downloads/ artifacts/
3. ./upload-images.sh artifacts/ sed.yaml
4. ./build-infrastructure.sh artifacts/ sed.yaml
5. ./deploy-vnflaf.sh artifacts/ sed.yaml
6. ./provision-vnflaf.sh artifacts/ sed.yaml
7. ./install-enm.sh sed.yaml

In order to run steps 3-5, you have to source your keystonerc file first, so
that your Openstack credentials are in your environment.

The one thing these scripts don't do for you is generated your "sed.yaml" file.
Currently, you have to use an Excel spreadsheet to create that. Once you have
it, though, the `upload_images.sh` script substitutes the current image names
into it for you.

Future to-do list
-----------------

In order to have a completely automated ENM install, the following things must
also be done. There are comments referring to these steps in the code as
well.

- expect script to log in and change the cloud-user password on the vnf-lcm
  services instance so that we can log in
- scp the deployment workflows RPM and install it
- create the /vnflcm-ext/enm folder and set the correct permissions
- transfer the sed.json into that directory and set permissions
- trigger ENM install using the VNF REST API

