#!/bin/bash
#
# This runs a crux build and cloud test using docker-machine.
#

# Where are we?
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Move to just outside the crux repo.
cd "$script_dir"/../..
# Make sure it's actually the crux repo.
if [ ! -d crux ] ; then
    echo "This script has to be inside crux/tools."
    echo "You can't move it elsewhere without making some changes to it."
    exit 1
fi

node_name=$USER-crux-build

function cleanup {
    docker-machine rm -f -y $node_name
}
trap cleanup EXIT

# Create docker-machine
docker-machine create --driver digitalocean --digitalocean-image $ERIX_NODE_ID --digitalocean-access-token $DIGITALOCEAN_TOKEN $node_name

# transfer code to docker-machine
tar cf - crux | docker-machine ssh $node_name 'tar xvf -'

# build crux
docker-machine ssh $node_name "cd crux && make container"

# Set env to point docker to the docker machine
eval $(docker-machine env $node_name)

# Run test.
bash crux/tests/cloud/basic/test.sh
