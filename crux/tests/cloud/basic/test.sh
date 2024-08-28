#!/bin/bash
#
# A basic cloud test for crux in digitalocean.
#

# Get the directory that this script is currently in.
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
# Import some relative paths.
source $SCRIPT_DIR/../cloud-test-lib.sh
source $SCRIPT_DIR/config.sh

# Figure out the image id of the docker image we'll be using.
IMAGE_ID=$( docker images -qf reference="$DOCKER_IMAGE" )
: ${IMAGE_ID:?"Can't find ID for $DOCKER_IMAGE -- can't continue."}
IMAGE_NAME=$TEST_NAME

main () {
    # 1. Check for dependencies.
    check_dependencies || return $?

    # 2. Create the cluster, set up docker swarm, and create registry service.
    create_swarm_cluster "$CLUSTER_SIZE" || return $?

    # 3. Copy our image to the registry.
    push_image_to_registry "$IMAGE_ID" "$IMAGE_NAME" || return $?

    # 4. Create an overlay network for flocking.
    create_overlay_network "$TEST_NAME" 10.20.30.0/24 || return $?

    # Variables used for the swarm services.
    flock_key="27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf"
    beacon_port="29718"
    network_option="--network $TEST_NAME"

    # NOTE: this is single-quoted so that it gets passed as text as part of the
    # swarm commands below; it shouldn't be evaluated until it gets inside the
    # container. This means we also have to wrap the service commands in an
    # "sh -c" in order to make sure this gets evaluated properly.
    detect_ip='$(grep `hostname` /etc/hosts | awk "{print \$1}" )'

    # 5. Create the beacon service.
    # NOTE: we can't use "beacon" in the --beacon option for this because if we
    # do, we'll try to attach to the docker service IP, which won't work.
    # NOTE: this has to be inside of a while loop, because it quits as soon as
    # it detects any sort of flock, and if we let swarm restart it, it won't
    # restart fast enough.
    watch_command="/bin/sh -c 'while true; do fulcrum watch --beacon $detect_ip:$beacon_port --key $flock_key --n $REPLICAS; sleep 1; done'"
    create_swarm_service "beacon" "$IMAGE_NAME" "1" "$network_option" "$watch_command" || return $?

    # 6. Create the flock service.
    # NOTE: since we use a name instead of an IP to contact the beacon, if the
    # beacon service isn't started before the flock, the DNS name won't exist,
    # and the flock processes will exit with:
    # dialing() failed: bad ip string 'beacon' (lookup beacon on 127.0.0.11:53:
    # no such host)
    flock_command="/bin/sh -c 'fulcrum flock --strew --beacon beacon:$beacon_port --key $flock_key --ip $detect_ip'"
    create_swarm_service "flock" "$IMAGE_NAME" "$REPLICAS" "$network_option" "$flock_command" || return $?

    # 7. Run test here to verify that the service came up properly.
    # NOTE: the "flocks" message will show up as soon as there's a flock, and
    # the beacon will exit. This is true even if the number of members is less
    # than what's specified in the --n switch. To work around this, we grep for
    # "n=$REPLICAS".
    while true ; do
        flock_string=$(get_service_logs "beacon" | grep "flocks" | tail -1 | grep "Stable:true" | grep "n=$REPLICAS")
        if [ $? -eq 0 ] ; then
            echo "Flock is stable! Flock string: $flock_string"
            return 0
        else
            echo "Flock isn't stable yet, trying again..."
            sleep 2
        fi
    done

    return 0
}

main

# The cloud-test-lib should clean up the docker-machine nodes automatically, so
# we don't have to do it here.
