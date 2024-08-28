#!/bin/bash

# Install kubernetes with kubeadm.
# This package doesn't build a cluster, it just installs the components
# necessary to do so.

set -e

echo "Disabling swap, and commenting out swap line in /etc/fstab..."
sudo swapoff -a
sudo sed -i.bak '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

echo "Passing bridged traffic to iptables chains (required for CNI plugins)..."
sudo modprobe br_netfilter
sudo sysctl net.bridge.bridge-nf-call-iptables=1
echo "br_netfilter" | sudo tee -a /etc/modules
echo "net.bridge.bridge-nf-call-iptables=1" | sudo tee -a /etc/sysctl.conf

install_docker () {
    echo "Installing needed support packages..."
    sudo apt -qy update
    sudo apt -qy install ebtables ethtool apt-transport-https
    sudo apt -qy install docker.io
    sudo systemctl enable docker.service
    sudo systemctl start docker.service
}
until install_docker
do
    echo "Trying again. Thanks, Ubuntu..."
done

echo "Adding kubernetes repository..."
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
cat <<EOF | sudo tee /etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF

install_kubernetes () {
    echo "Installing kubernetes tools..."
    sudo apt -qy update
    sudo apt -qy install kubelet kubeadm kubectl
}
until install_kubernetes
do
    echo "Trying again. Thanks, Ubuntu..."
done

echo "Done!"
