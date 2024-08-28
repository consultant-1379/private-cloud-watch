#!/bin/bash

echo "You probably shouldn't be running this, unless Terraform has totally"
echo "gone out to lunch and you need to 'start fresh'."
echo "Also, don't expect it to work very well."
echo
read -p "Are you sure you want to DESTROY the head node and associated resources? (y/n)" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]] ; then
    echo "Phew!"
    exit 1
fi
echo "Ok, attempting to destroy everything! 5 seconds to change your mind..."
sleep 5
echo "Here we go!"

openstack server delete headnode
openstack security group delete headnode
openstack keypair delete headnode
openstack port list -f value --router=terraform | cut -d\  -f1 | while read uuid ; do
    $openstack port delete $uuid
done
openstack router delete terraform
openstack network delete terraform

echo "Please figure out whether this head node had any floating ips attached,"
echo "and release them. I haven't figured out how to do this via command line yet."
echo
echo "Also, you'll probably have to delete the ports, router, and network via the GUI,"
echo "because of the \"failed to delete port\" error."
