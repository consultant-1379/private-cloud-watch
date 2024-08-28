#!/bin/bash
#
# This runs a crux build and test using docker-machine.
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

# Install some dependencies
go_archive='https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz'
docker-machine ssh $node_name "cd /usr/local ; curl $go_archive | tar zxf - "
docker-machine ssh $node_name "apt-get update && apt-get install -y autoconf build-essential ca-certificates curl gawk git libtool pkg-config unzip wget yasm"

# transfer code to docker-machine
project_dir='~/go/src/github.com/erixzone'
docker-machine ssh $node_name "mkdir -p $project_dir"
tar cf - crux | docker-machine ssh $node_name "cd $project_dir ; tar xvf -"

# build and test crux
docker-machine ssh $node_name "export PATH=$PATH:/usr/local/go/bin:~/go/bin ; export GOPATH=/root/go ; cd $project_dir/crux && make && make test"
