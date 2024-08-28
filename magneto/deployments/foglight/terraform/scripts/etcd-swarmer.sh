#!/bin/bash
#
# Creates an etcd which is used by swarmer when building a docker swarm cluster.

version="v3.2.9"
url="https://github.com/coreos/etcd/releases/download"
dest_dir="/usr/local/bin"
envfile="/etc/default/etcd"
systemd="/etc/systemd/system"
data_dir="/var/run/etcd"

etcd_ip=$1 ; shift
etcd_port=$1 ; shift
options=$@

set -eu

if [ -z "$etcd_ip" ] || [ -z "$etcd_port" ]; then
    echo "Usage: $0 <etcd ip> <etcd port> <optional args>"
    exit 1
fi

echo "Downloading etcd..."
filename="etcd-${version}-linux-amd64.tar.gz"
curl -L ${url}/${version}/${filename} -o /tmp/${filename}
sudo tar xzvf /tmp/${filename} -C ${dest_dir} --strip-components=1 --wildcards */etcd */etcdctl

${dest_dir}/etcd --version
${dest_dir}/etcdctl --version

sudo mkdir -p ${data_dir}

echo "Creating etcd environment file..."
cat <<EOF | sudo tee ${envfile}
ETCD_LISTEN_CLIENT_URLS="http://${etcd_ip}:${etcd_port}"
ETCD_ADVERTISE_CLIENT_URLS="http://${etcd_ip}:${etcd_port}"
ETCD_DATA_DIR="${data_dir}"
OPTIONS="${options}"
EOF

echo "Creating etcd systemd service..."
cat <<EOF | sudo tee ${systemd}/etcd.service
[Unit]
Description=Etcd key-value store
After=network.target

[Service]
EnvironmentFile=-${envfile}
ExecStart=/usr/local/bin/etcd \$OPTIONS

[Install]
WantedBy=multi-user.target
EOF

echo "Starting etcd service..."
sudo systemctl daemon-reload
sudo systemctl enable etcd
sudo systemctl start etcd

echo "Checking to make sure etcd is alive..."
sleep 5
sudo systemctl is-active --quiet etcd
