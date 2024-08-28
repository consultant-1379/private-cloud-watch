#!/bin/bash
#
# Install the vault binary.

vault_zip="https://releases.hashicorp.com/vault/0.9.6/vault_0.9.6_linux_amd64.zip"
target="/usr/local/bin"

scratch=$(mktemp -d -t vault-client.XXXXX)
function cleanup {
    rm -rf "$scratch"
}
trap cleanup EXIT

echo "Installing dependencies..."
sudo apt -qy update && sudo apt -qy install wget unzip

echo "Installing vault client into ${target}..."
cd $scratch
wget $vault_zip
cd $target
sudo unzip -o $scratch/*
