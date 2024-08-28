#!/bin/bash
#
# Install the most recent stable docker and run hello-world.

set +e
sudo apt -qy remove docker docker-engine docker.io
set -e
sudo apt -qy update && \
    sudo apt -qy install apt-transport-https ca-certificates curl \
        software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"

sudo apt -qy update && \
    sudo apt -qy install docker-ce

sudo docker run hello-world
