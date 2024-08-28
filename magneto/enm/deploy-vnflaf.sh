#!/bin/bash
#
# When pointed at an ENM artifacts directory, this script deploys VNF-LAF.
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

# Create the LAF DB volume.
stack_name="${deployment_id}_LAF_DB_Volume"
heat_template="${artifacts}/merged_volumes/vnflafdb_bs.yaml"
create_stack "$stack_name" "$heat_template" || exit 1

# Check to make sure that the volume status is "available".
vnflaf_db_volume="vnflafdb_volume"
wait_for_volume "$vnflaf_db_volume" || exit 1

# Create the VNF-LAF DB stack (ipv4 only version).
stack_name="${deployment_id}_VNF_LAF_DB"
heat_template="${artifacts}/merged_applications/vnflaf_db.yaml"
volume_uuid=$(get_volume_id "$vnflaf_db_volume")
create_stack "$stack_name" "$heat_template" "vnflafdb_volume_uuid=${volume_uuid}" || exit 1

# The following two appear to be deprecated as of 12/19/2017.
## Create the LAF ENM volumes.
#stack_name="${deployment_id}_LAF_ENM_Volume"
#heat_template="${artifacts}/merged_volumes/vnflaf_enm_iso_bs.yaml"
#iso_volume="vnflaf_enm_isovol"
#create_stack "$stack_name" "$heat_template" || exit 1
#wait_for_volume "$iso_volume" || exit 1
#
#stack_name="${deployment_id}_LAF_BS_Stack"
#heat_template="${artifacts}/merged_volumes/laf_bs.yaml"
#laf_volume="laf_volume"
#create_stack "$stack_name" "$heat_template" || exit 1
#wait_for_volume "$laf_volume" || exit 1

# Create VNF LAF services (ipv4 only version).
#enm_iso_volume_uuid=$(get_volume_id "$iso_volume")
#laf_volume_uuid=$(get_volume_id "$laf_volume")
stack_name="${deployment_id}_VNF_LAF_SERVICES"
heat_template="${artifacts}/merged_applications/vnflaf.yaml"
#create_stack "$stack_name" "$heat_template" "enm_iso_volume_uuid=${enm_iso_volume_uuid}" "laf_volume_uuid=${laf_volume_uuid}" || exit 1
create_stack "$stack_name" "$heat_template" || exit 1

echo "The manual says you should wait 15-20 minutes before attempting to
connect to the VNF-LAF Services web UI.

I've put a 20 minute sleep here. Nothing happens afterwards, so you're
safe to Ctrl-C if you need to.

Good luck."

sleep 1200
