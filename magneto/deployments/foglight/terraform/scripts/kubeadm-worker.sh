#!/bin/bash

api_server=$1 ; shift
api_port=$1 ; shift
join_token=$1 ; shift
arguments=$@

set -eu

if [ -z "$api_server" ] || [ -z "$api_port" ] || [ -z "$join_token" ] ; then
    echo "Usage: $0 <api server> <api port> <join token> <optional args>"
    exit 1
fi

echo "Running kubeadm join..."
sudo kubeadm join --token "$join_token" ${api_server}:${api_port} --discovery-token-unsafe-skip-ca-verification

echo "Setting up kubectl client..."
mkdir -p $HOME/.kube
# Wait until the file exists before trying to copy it.
until [ -f /etc/kubernetes/kubelet.conf ]
do
    sleep 1
done
sudo cp -i /etc/kubernetes/kubelet.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
