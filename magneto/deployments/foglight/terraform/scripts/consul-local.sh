#!/bin/bash
#
# Create a one-node consul server that only runs on localhost.
#
# I made this to create a backend for a one-node Vault.
#
# Running a local Consul could be a good idea if you are interested in database
# stability and avoiding data corruption. It obviously doesn't provide high
# availability.
#
# This script runs Consul in a container, restoring its data from a snapshot in
# an object store if it can find one. It also sets up a job to snapshot the
# database every 4 hours and upload the snapshot to object store using the
# "snapshot-consul.sh" script.

service_name=$1
consul_dir="/srv/consul"

# Make consul directories.
function make_consul_dirs () {
    echo "Making consul directories at $consul_dir"
    for i in "config" "data" "snapshot" ; do
        mkdir -p "$consul_dir/$i" || return $?
    done
}

# Start consul docker container and wait until it's ready.
function start_consul () {
    echo "Starting consul container"
    docker run --name $service_name \
        --detach --net=host --restart=always \
        --mount type=bind,source=$consul_dir,target=/consul \
        consul agent -server -bind=127.0.0.1 -bootstrap-expect=1 \
        || return $?
    container=$(docker ps -qf name=$service_name)

    echo "Waiting for consul leader election"
    leader=""
    while [ -z "$leader" ] ; do
        leader=$(docker exec "$container" consul info | \
            grep leader_addr | grep 127.0.0.1)
        sleep 1
    done
    echo "Localhost elected as leader"
    return 0
}

# Pull data from snapshot if we can find one.
function restore_consul_from_snapshot () {
    echo "Pulling most recent $service_name snapshot from spaces"
    # Sort by creation time, then take the most recent one.
    filename=$(s3cmd ls s3://$service_name/ | \
        grep -v '\/$' | sort | tail -n1 | awk '{print $4}' | sed 's/.*\///g')

    # If we got one, load it in.
    if [ ! -z "$filename" ]; then
        echo "Loading snapshot into running consul"
        s3cmd get "s3://$service_name/$filename" || return $?
        backup_dir="$consul_dir/snapshot"
        mv "$filename" "$backup_dir" || return $?
        docker exec "$container" consul snapshot restore "/consul/snapshot/$filename" || return $?
    fi

    return 0
}

# Create cronjob that backs up consul db to spaces.
function make_snapshot_cronjob () {
    echo "Creating cronjob that snapshots consul every 4 hours"
    job_name="snapshot-consul"
    chmod 700 /usr/local/bin/$job_name.sh
    cat << EOF > /etc/cron.d/$job_name
0 */4 * * * root /usr/local/bin/$job_name.sh $service_name >/var/log/$job_name.out 2>&1
EOF
}

function main () {
    echo "Installing $service_name"
    make_consul_dirs || exit 1
    start_consul || exit 1
    restore_consul_from_snapshot || exit 1
    make_snapshot_cronjob || exit 1
    echo "Done!"
    exit 0
}

main
