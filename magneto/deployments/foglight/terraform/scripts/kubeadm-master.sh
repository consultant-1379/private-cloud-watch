#!/bin/bash

kube_router_url="https://raw.githubusercontent.com/cloudnativelabs/kube-router/master/daemonset/kubeadm-kuberouter.yaml"

api_port=$1 ; shift
join_token=$1 ; shift
arguments=$@

set -eu

if [ -z "$api_port" ] || [ -z "$join_token" ] ; then
    echo "Usage: $0 <api port> <join token> <optional args>"
    exit 1
fi

echo "Running kubeadm init..."
sudo kubeadm init --apiserver-bind-port $api_port --token "$join_token" $arguments

echo "Setting up kubectl client..."
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

echo "Setting up kube-router networking..."
sudo kubectl apply -f "$kube_router_url"

echo "Done!"
