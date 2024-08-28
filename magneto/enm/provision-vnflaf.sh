#!/bin/bash
#
# This script provisions a VNF-LAF services instance. It's meant to be
# run after the deploy-vnflaf.sh script.
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

# Add a router interface for the external network that's been added.
# This is very foglight-specific, so it probably shouldn't be here, but
# it's here right now for the sake of completion.

# Add a floating IP for the VNF services instance so that we can get to it.
# Again, this is environment-specific.
# At the end of this process we need the floating IP.

# Change the cloud-user password on the node using an expect script.
sudo apt -qy install expect
expect "$script_directory"/set-user-password.exp $floating_ip

# Transfer the ERICenmdeploymentworkflow RPM to the VNF-LAF services instance.

# Install the RPM.

# Create the /vnflcm-ext/enm folder if it isn't already present, and set
# permissions appropriately.

# Convert the YAML SED file to JSON.
json_sed_file=$(echo "$sed_file" | sed 's/yaml$/json/')
python -c 'import sys, yaml, json; json.dump(yaml.load(sys.stdin), sys.stdout, indent=4)' < "$sed_file" > "$json_sed_file"

# Transfer the sed.json file and set permissions.

