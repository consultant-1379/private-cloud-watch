#!/bin/bash
#
# When pointed at an ENM artifacts directory, this script creates a security
# group and an internal network in preparation for a VNF-LAF deployment.
#
# If you run it and it fails due to openstack credentials, you'll need to
# source the relevant credentials script, which is available via the openstack
# gui.
# (sidebar: project -> compute -> access -> security, tab: api access).
#

# Import openstack functions.
script_directory=$(dirname $0)
source "$script_directory"/openstack-functions.sh

# Collect and test arguments.
artifacts=$1 ; shift
sed_file=$1 ; shift

set -u

if [ -z "$artifacts" ] || [ -z "$sed_file" ] ; then
    echo "Usage: $0 <artifacts dir> <SED (heat environment) file>"
    exit 1
fi

if [ ! -d "$artifacts" ] ; then
    echo "Must pass a directory containing ENM artifacts!"
    exit 1
fi

if [ ! -f "$sed_file" ] ; then
    echo "Must pass a SED (heat environment) yaml file!"
    exit 1
fi

artifacts=$(readlink -m "$artifacts")
sed_file=$(readlink -m "$sed_file")

# Install dependencies.
install_openstack_dependencies || exit 1

# Get deployment id.
deployment_id=$(get_deployment_id "$sed_file")
if [ $? -ne 0 ] ; then
    echo "Couldn't parse deployment id from SED file!"
    exit 1
fi

# Ensure that the ENM keypair exists and is in openstack.
ensure_enm_keypair || exit 1

# Create security group stack.
stack_name="${deployment_id}_network_security_group"
heat_template="${artifacts}/infrastructure_resources/network_security_group_stack.yaml"
create_stack "$stack_name" "$heat_template" || exit 1

# Create internal network stack.
stack_name="${deployment_id}_network_internal_stack"
heat_template="${artifacts}/infrastructure_resources/network_internal_stack.yaml"
create_stack "$stack_name" "$heat_template" || exit 1

# Create experimental external network stack.
# I added the heat template and custom SED fields that create this, because the
# ENM install assumes this already exists and was created by admins or
# something. I'd use the ENM network in Foglight, but it doesn't meet the
# requirements (it's not DHCP, for example).
stack_name="${deployment_id}_network_external_stack"
heat_template="${script_directory}/network_external_stack.yaml"
create_stack "$stack_name" "$heat_template" || exit 1

# If the stacks already exist, even if we failed to fully create their
# resources the last time we ran, these commands will fail with a "stack
# already exists" error. We may choose to do an "openstack stack delete <stack
# name>" automatically, but right now we don't...
