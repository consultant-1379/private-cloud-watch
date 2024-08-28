#!/bin/bash

digitalocean_api="https://api.digitalocean.com/v2"

# Try not to leave nodes hanging around.
trap cleanup EXIT

cleanup () {
    destroy_swarm_cluster
    exit
}

# Check for local dependencies.
check_dependencies () {
    # Check for environment variables.
    : ${TEST_NAME:?"The TEST_NAME variable is empty, can't continue."}
    : ${DIGITALOCEAN_TOKEN:?"The DIGITALOCEAN_TOKEN variable is empty, can't continue."}

    # TEST_NAME can't contain spaces.
    case ${TEST_NAME} in
        *[[:space:]]* ) echo "TEST_NAME contains spaces, which isn't allowed; can't continue." && return 3 ;;
    esac

    # We have a good test_name, so we can create the cluster name from it.
    generate_cluster_name || return 4

    # Check for binaries... docker, docker-machine.
    dependencies="docker docker-machine python"
    for dep in $dependencies ; do
        if ! which $dep >/dev/null ; then
            echo "Can't find '$dep' executable in path, can't continue."
            return 2
        fi
    done

    echo "Preflight check passed."
}

# Create cluster name from TEST_NAME and username.
generate_cluster_name () {
    local user_name="$( whoami )"
    # If user_name is blank, just use 'unknown'.
    : "${user_name:="unknown"}"
    local random=$( od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3}' )
    cluster_name="$user_name-test-$random-$TEST_NAME"
    echo "Using cluster name: $cluster_name"
    return 0
}

# Create a cluster of a given size, then start docker swarm
# and docker registry.
create_swarm_cluster () {
    cluster_size="$1"
    ((cluster_size < 1 || cluster_size > 16)) \
        && echo "cluster_size needs to be an integer between 1 and 16." && exit 1

    # Check for previously existing machines with this name. (Possibly
    # also delete them?)
    clean_preexisting_nodes || return $?

    # Remove any remnants of old nodes that might be in docker-machine.
    remove_old_docker_machine_remnants || return $?

    # Create cluster of given size.
    create_cluster || return $?

    # Start docker swarm on manager.
    start_swarm_manager || return $?

    # Join swarm on all other nodes.
    start_swarm_workers || return $?

    # Start docker registry service inside swarm.
    start_docker_registry || return $?

    return 0
}

# Clean up preexisting nodes that have the name we're going to use.
clean_preexisting_nodes () {
        local output status
        output=$( get_digitalocean_nodes )
        status=$?
        if [ $status -ne 0 ] ; then
            echo "Error $status when attempting to get node list, won't continue."
            return 1
        fi
        echo "$output" | while read node
        do
            if [[ "$node" == "${cluster_name}"* ]] ; then
                echo "Preexisting node found: $node"
                case "${REMOVE_PREEXISTING_NODES}" in
                    yes)
                        remove_digitalocean_node "$node" || return $?
                        ;;
                    *)
                        echo "Ignoring this and attempting to move forward anyway."
                        ;;
                esac
            fi
        done
        return 0
}

# Output a list of nodes that currently exist in digitalocean.
get_digitalocean_nodes () {
    read -r -d '' python_command <<EOF
import sys, json
data = json.load(sys.stdin)
for element in data['droplets']:
    print "%s: droplet ID %s" % (element['name'], element['id'])
EOF
    local output
    output=$( curl -X GET "$digitalocean_api/droplets" \
        -H "Authorization: Bearer $DIGITALOCEAN_TOKEN" 2>/dev/null )
    if [ $? -ne 0 ] ; then
        return 1
    fi
    # The output of this python command is the stdout of this function.
    echo "$output" | python -c "$python_command" 2>/dev/null
    if [ $? -ne 0 ] ; then
        return 2
    fi
}

# Remove a given digitalocean node. The node argument must include the droplet
# id in the 4th field.
remove_digitalocean_node () {
    local node="$1"
    local droplet_id=$( echo "$node" | awk '{print $4}' )
    echo "Attempting to remove preexisting droplet: $droplet_id"
    status=$( curl -s -o /dev/null -w "%{http_code}" \
        -X DELETE "$digitalocean_api/droplets/$droplet_id" \
        -H "Authorization: Bearer $DIGITALOCEAN_TOKEN" )
    case $status in
        204)
            echo "Successfully removed droplet $droplet_id"
            return 0
            ;;
        404)
            echo "Droplet $droplet_id not found!"
            return 1
            ;;
        *)
            echo "Response code $status when removing droplet $droplet_id"
            return 1
            ;;
    esac
}

# Remove any remnants of nodes in docker machine that start with our cluster
# name. This is potentially destructive if someone happens to name their
# manually-created nodes in the same way that we name nodes, which is unlikely.
remove_old_docker_machine_remnants () {
    echo "Removing any remnants of old docker-machine nodes."
    seq 1 $cluster_size | xargs -n1 -P16 -I{} docker-machine rm -f -y \
        ${cluster_name}-{} >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Got errors removing stale node entries from docker-machine. Try manual cleanup."
        return 1
    fi
    return 0
}

# Create the nodes we'll use for our test. Uses xargs to run docker-machine in
# parallel.
create_cluster () {
    echo "Creating $cluster_name cluster of $cluster_size nodes. Please wait..."
    seq 1 $cluster_size | xargs -n1 -P16 -I{} docker-machine create \
        --driver digitalocean \
        --digitalocean-access-token $DIGITALOCEAN_TOKEN \
        ${cluster_name}-{} >/dev/null
    if [ $? -ne 0 ]; then
        echo "Got errors creating cluster using docker-machine. Try again in a few minutes."
        return 1
    fi
}

# Run docker swarm init on node 1.
start_swarm_manager () {
    manager="${cluster_name}-1"
    echo "Starting swarm manager on $manager."
    local manager_ip=$( docker-machine ip $manager )
    local output=$( docker-machine ssh $manager \
        "docker swarm init --advertise-addr=$manager_ip" )
    echo "$output" | grep "is now a manager"
    if [ $? -ne 0 ] ; then
        echo "Couldn't create swarm! \n${output}"
        return 1
    fi
}

# Run docker swarm join on all nodes other than node 1.
start_swarm_workers () {
    local worker_join_command=$( docker-machine ssh $manager \
        "docker swarm join-token worker" | grep "docker swarm join" )
    for i in `seq 2 $cluster_size` ; do
        local worker=${cluster_name}-${i}
        echo "Joining swarm with node $worker."
        local output=$( docker-machine ssh $worker "$worker_join_command" )
        echo "$output" | grep "node joined a swarm"
        if [ $? -ne 0 ] ; then
            echo "Couldn't join swarm with node $worker! \n${output}"
            return 1
        fi
    done
}

# Start a docker registry service inside the swarm.
start_docker_registry () {
    echo "Starting docker registry service inside swarm."
    local output=$( docker-machine ssh $manager \
        'docker service create --name registry --publish 5000:5000 registry:2' )
    echo "$output" | grep "Service converged"
    if [ $? -ne 0 ] ; then
        echo "Couldn't start registry service!  \n${output}"
        return 1
    fi
}

# Push a given docker image id from the local node to the swarm registry.
push_image_to_registry () {
    local image_id=$1
    local image_name=$2
    echo "Pushing $image_name image to swarm registry."
    # Copy our image to the manager.
    local output
    output=$( docker save $image_id | \
        docker-machine ssh $manager "docker load" )
    echo "$output" | grep "Loaded image"
    if [ $? -ne 0 ] ; then
        echo "Couldn't load image!"
        return 1
    fi
    # Tag image on the manager.
    docker-machine ssh $manager "docker tag $image_id $image_name"
    if [ $? -ne 0 ] ; then
        echo "Couldn't tag image!"
        return 1
    fi
    # Push our image to the swarm registry.
    output=$( docker-machine ssh $manager \
        "docker tag $image_name localhost:5000/$image_name && \
        docker push localhost:5000/$image_name" )
    echo "$output" | tail -1 | grep latest
    if [ $? -ne 0 ] ; then
        echo "Couldn't push image to swarm registry!"
        return 1
    fi
    return 0
}

# Create an overlay network.
create_overlay_network () {
    local net_name=$1
    local subnet=$2
    echo "Creating overlay network $net_name using subnet $subnet"
    docker-machine ssh $manager \
        "docker network create --driver overlay --subnet $subnet $net_name"
    if [ $? -ne 0 ] ; then
        echo "Couldn't create overlay network!"
        return 1
    fi
}

# Create a service given an image, a replica count, and some runtime options.
create_swarm_service () {
    local service_name=$1
    local image_name=$2
    local replicas=$3
    local options=$4
    local cmd=$5
    ((replicas < 1 || replicas > 128)) && \
        echo "replicas (arg 3) needs to be an integer between 1 and 128." && exit 1
    echo "Creating $replicas replicas of $service_name service."
    docker-machine ssh $manager \
        "docker service create --quiet --name $service_name --replicas=$replicas $options localhost:5000/$image_name $cmd"
    if [ $? -ne 0 ] ; then
        echo "Couldn't create swarm service!"
        return 1
    fi
}

# Print the logs for a given service.
get_service_logs () {
    local service_name=$1
    docker-machine ssh $manager \
        "docker service logs $service_name"
}

# Destroy a swarm cluster.
destroy_swarm_cluster () {
    echo "Destroying swarm cluster."
    for i in `seq 1 $cluster_size` ; do
        local node="${cluster_name}-${i}"
        local output
        output=$( docker-machine rm -f -y $node 2>/dev/null)
        echo "$output" | grep "Successfully"
        if [ $? -ne 0 ] ; then
            echo "Couldn't remove node $node!"
            # Try the others even if this failed, don't return yet.
        fi
    done
    return 0
}
