#!/bin/bash
#
# Start a one-node Vault server in a docker container pointing at a local
# consul.
#
# Note: it's not considered a best practice to run Vault in a docker container,
# nor is it considered best practice to run it on a VM or in the cloud.

echo "Starting vault"

service_name="vault-server"
vault_dir="/srv/vault"
certs_dir="$vault_dir/certs"
config_dir="$vault_dir/config"
vault_config=$(cat << "EOF"
storage "consul" {
  address = "127.0.0.1:8500"
  path = "vault"
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_cert_file = "/vault/certs/host.chain.crt"
  tls_key_file = "/vault/certs/host.key"
  tls_client_ca_file = "/vault/certs/erixzone-ca-root.crt"
}
EOF
)

function make_dirs () {
    echo "Making vault directories at $vault_dir"
    for i in "$certs_dir" "$config_dir" ; do
        mkdir -p "$i" || return $?
    done
}

function copy_certs () {
    echo "Copying certs to $certs_dir"
    # Vault expects you to concatenate the host cert and the intermediate ca.
    # Rather than try to do this here, I expect it's already provided as a
    # "chain.crt" file.
    cp "/etc/ssl/private/host.chain.crt" $certs_dir || return $?
    cp "/etc/ssl/private/host.key" $certs_dir || return $?
    cp "/usr/local/share/ca-certificates/erixzone-ca-root.crt" $certs_dir || return $?
}

function write_config () {
    echo "Writing vault config to $config_dir"
    echo "$vault_config" >"${config_dir}/config.json" || return $?
}

function start_vault () {
    echo "Starting vault container"
    docker run --name "$service_name" \
        --detach --net=host --restart=always \
        --cap-add=IPC_LOCK \
        --mount type=bind,source=$certs_dir,target=/vault/certs \
        --mount type=bind,source=$config_dir,target=/vault/config \
        vault server \
        || return $?

    docker logs $(docker ps -qf name=$service_name)
}

function main () {
    make_dirs || exit 1
    copy_certs || exit 1
    write_config || exit 1
    start_vault || exit 1
    echo "Done!"
    exit 0
}

main

