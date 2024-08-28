#!/bin/bash

# Function to install openstack CLI dependencies.
install_openstack_dependencies () {
    echo "Installing openstack CLI dependencies."
    umask 022
    sudo pip -q install python-openstackclient python-heatclient
    return $?
}

# Function to parse deployment ID from sed file.
get_deployment_id () {
    local sed_file=$1
    # The regexp removes all whitespace. There shouldn't be any other than a
    # space at the beginning...
    deployment_id=$(grep "deployment_id:" "$sed_file" | cut -d: -f2 | sed 's/\s*//g')
    echo "$deployment_id"
    if [ "$deployment_id" == "" ]; then
        return 1
    fi
    return 0
}

# Function to check for stack pre-existence.
check_stack_existence () {
    local stack_name=$1
    echo "Testing for pre-existence of stack $stack_name ..."
    set +e
    openstack stack show $stack_name >/dev/null 2>&1
    exists=$?
    set -e
    if [ $exists -eq 0 ] ; then
        echo "Stack $stack_name seems to already exist!"
        echo -e "If you want to recreate it, run:\n\tyes | openstack stack delete $stack_name\nThen wait a few minutes, and run this again."
        return 1
    fi
    return 0
}

# Function to create a stack.
run_create_stack_command () {
    local stack_name="$1" && shift
    local heat_template="$1" && shift
    local parameters="$@"
    # Build parameters list, if there are any parameters.
    local parameters_list=""
    for param in ${parameters[@]} ; do
        parameters_list="$parameters_list --parameter $param"
    done
    echo "Creating stack: $stack_name"
    openstack stack create \
        -e "$sed_file" \
        -t "$heat_template" \
        $parameters_list \
        $stack_name
    if [ $? -ne 0 ]; then
        echo "Couldn't create stack $stack_name!"
        return 1
    fi
    return 0
}

get_stack_status () {
    local stack_name=$1
    local status=$(openstack stack show $stack_name --column stack_status --format value)
    local return=$?
    echo "$status"
    return $return
}

get_stack_status_reason () {
    local stack_name=$1
    local status=$(openstack stack show $stack_name --column stack_status_reason --format value)
    local return=$?
    echo "$status"
    return $return
}

# Function to wait until stack creation succeeds or errors out.
wait_for_stack_success () {
    local stack_name=$1
    echo "Waiting for creation of stack $stack_name."
    local stack_status="CREATE_IN_PROGRESS"
    local i=0
    while [ "$stack_status" == "CREATE_IN_PROGRESS" ] ; do
        stack_status=$(get_stack_status "$stack_name")
        i=$((i+1))
        if [ $i -gt 600 ] ; then
            echo "Timed out waiting for creation of stack $stack_name"
            return 1
        fi
        sleep 1
    done
    if [ "$stack_status" == "CREATE_COMPLETE" ] ; then
        echo "Successfully created stack $stack_name."
        return 0
    else
        echo "Problem creating stack $stack_name!"
        echo "Stack status: $stack_status"
        local stack_status_reason=$(get_stack_status_reason "$stack_name")
        echo "Reason: $stack_status_reason"
        return 1
    fi
}

# Function to get volume status.
get_volume_status () {
    local volume_name=$1
    local status=$(openstack volume show "$volume_name" --column status --format value)
    local return=$?
    echo "$status"
    return $return
}

# Function to get volume id.
get_volume_id () {
    local volume_name=$1
    local status=$(openstack volume show "$volume_name" --column id --format value)
    local return=$?
    echo "$status"
    return $return
}

# Function to wait for volume availability.
wait_for_volume () {
    local volume_name=$1
    echo "Waiting for volume $volume_name to become available."
    local volume_status=""
    local i=0
    while [ "$volume_status" != "available" ] ; do
        volume_status=$(get_volume_status "$volume_name")
        if [ "$volume_status" == "in-use" ] ; then
            echo "Volume $volume_name is in use, continuing."
            return 0
        fi
        i=$((i+1))
        if [ $i -gt 180 ] ; then
            echo "Timed out waiting for volume $volume_name"
            return 1
        fi
        sleep 1
    done
    echo "Volume $volume_name is available."
    return 0
}

# Meta-function to create a stack, with optional parameters.
create_stack () {
    local stack_name=$1 && shift
    local heat_template=$1 && shift
    local parameters=$@
    check_stack_existence "$stack_name"
    if [ $? -eq 1 ] ; then
        # If the stack already exists, continue on.
        return 0
    fi
    run_create_stack_command "$stack_name" "$heat_template" "$parameters"
    if [ $? -ne 0 ] ; then
        # If we couldn't create the stack, error out.
        return 1
    fi
    wait_for_stack_success "$stack_name"
}

# Ensure that the ENM keypair exists and is in openstack.
ensure_enm_keypair () {
    echo "Ensuring existence of ENM keypair."
    # Create ENM keypair, if it doesn't already exist.
    if [ ! -f ~/.ssh/id_rsa.enm ] ; then
        echo "Creating ENM ssh key."
        yes | ssh-keygen -f ~/.ssh/id_rsa.enm -t rsa -N ''
        if [ $? -ne 0 ] ; then
            return 1
        fi
    fi

    # Upload our keypair to openstack if there isn't already one there.
    # This is a bit cavalier, obviously.
    fingerprint=$(openstack keypair show enm --column fingerprint --format value)
    if [ "$fingerprint" == "" ] ; then
        openstack keypair create --public-key ~/.ssh/id_rsa.enm.pub enm
        return $?
    fi
}
