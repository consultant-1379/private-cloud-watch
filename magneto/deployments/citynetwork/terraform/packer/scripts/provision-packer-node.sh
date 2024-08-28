#!/bin/bash
#
# Provision a packer node.
#
# This is simplistic---it assumes that you'll log in as ubuntu and run packer
# to build your base images, then destroy this node and build a more
# full-featured environment using the new images that you built.

set -e

scratch=$(mktemp -d -t provision-packer-node.XXXXX)
function cleanup {
    rm -rf "$scratch"
}
trap cleanup EXIT

# Install dependencies.
sudo apt-get install -y python python-pip wget unzip

# Install ansible.
PATH="$HOME/.local/bin:$PATH"
python -m pip install --user --upgrade pip
pip install --user ansible

# Install packer.
cd "$scratch"
wget --quiet https://releases.hashicorp.com/packer/1.2.3/packer_1.2.3_linux_amd64.zip
sudo unzip -o -d /usr/local/bin "$scratch/packer*"
