#!/bin/bash

swarmer="./swarmer"
config="/etc/swarmer.yaml"
envfile="/etc/default/swarmer"
systemd="/etc/systemd/system"

etcd_server=$1 ; shift
etcd_port=$1 ; shift
role=$1 ; shift
arguments=$@

set -eu

if [ -z "$etcd_server" ] || [ -z "$etcd_port" ] || [ -z "$role" ]; then
    echo "Usage: $0 <etcd server> <etcd port> <role> <optional args>"
    exit 1
fi

if [ ! -f "${swarmer}" ] ; then
    echo "Can't find swarmer at ${swarmer}"
    exit 2
fi

echo "Moving swarmer to /usr/local/bin"
sudo mv ${swarmer} /usr/local/bin

echo "Creating swarmer config file..."
echo "etcd: http://${etcd_server}:${etcd_port}" | sudo tee ${config}
ismanager="false"
if [[ "${role}" = "manager" ]] ; then 
    ismanager="true"
fi
echo "manager: ${ismanager}" | sudo tee -a ${config}

echo "Creating swarmer environment file..."
echo "OPTIONS=${arguments}" | sudo tee ${envfile}

echo "Creating swarmer systemd service..."
cat <<EOF | sudo tee ${systemd}/swarmer.service
[Unit]
Description=Docker swarm mode clustering helper
After=network.target

[Service]
EnvironmentFile=-${envfile}
ExecStart=/usr/local/bin/swarmer \$OPTIONS

[Install]
WantedBy=multi-user.target
EOF

echo "Starting swarmer service..."
sudo systemctl daemon-reload
sudo systemctl enable swarmer
sudo systemctl start swarmer

echo "Checking to make sure swarmer is alive..."
sleep 5
sudo systemctl is-active --quiet swarmer
