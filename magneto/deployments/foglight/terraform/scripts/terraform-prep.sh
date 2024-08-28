#!/bin/bash -l
#
# Install terraform locally.

# Exit with error if any line fails.
set -e

# This only works on debian and ubuntu linux.
if [ ! -f /etc/debian_version ] ; then
    echo "This script only works on Debian and Ubuntu at the moment."
    echo
    echo "Feel free to update it for your OS."
    exit 1
fi

terraform="https://releases.hashicorp.com/terraform/0.11.7/terraform_0.11.7_linux_amd64.zip"
target="/usr/local/bin"

scratch=$(mktemp -d -t terraform-prep.XXXXX)
function cleanup {
    rm -rf "$scratch"
}
trap cleanup EXIT

echo "Installing dependencies..."
# We have to try this multiple times because digitalocean sucks.
set +e
success=1
while [ $success -ne 0 ]; do
    sudo apt -qy update && sudo apt -qy install wget zip unzip python-pip
    success=$?
done
set -e

# Create terraform ssh key if it doesn't already exist.
if [ ! -f ~/.ssh/id_rsa.terraform ] ; then
    echo "Creating terraform ssh key..."
    yes | ssh-keygen -f ~/.ssh/id_rsa.terraform -t rsa -N ''
fi

# Download terraform.
echo "Installing terraform into ${target}..."
cd $scratch
wget $terraform
cd $target
sudo unzip -o $scratch/*

# Install openstack commands.
#echo "Installing openstack commands..."
#umask 022
#sudo pip -q install python-openstackclient python-swiftclient python-heatclient
