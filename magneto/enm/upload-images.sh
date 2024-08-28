#!/bin/bash
#
# When pointed at an ENM artifacts directory, this script uploads the necessary
# images to Openstack. It also needs to update a SED (heat environment data)
# file with the image names that it uploads.
#
# If you run it and it fails due to openstack credentials, you'll need to
# source the relevant credentials script, which is available via the openstack
# gui.
# (sidebar: project -> compute -> access -> security, tab: api access).
#

# Collect and test arguments.
artifacts=$1 ; shift
sed_file=$1 ; shift

set -eu

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

# The filenames include version numbers, but we don't necessarily know them, so
# we use these strings to match filenames.
declare -a partial_filenames=(
    "ERICenm"
    "ERICrhel6baseimage"
    "ERICrhel6jbossimage"
    "ERICrhel7baseimage"
    "ERICrhelpostgresimage"
    "ERICrhelvnflafimage"
    "httpd_createrepo"
    "RHEL_Media"
    "RHEL_OS_Patch_Set"
)

# Make sure there's one for each entry in partial_filenames above.
declare -A variable_names=(
    ["ERICenm"]="enm_iso_image"
    ["ERICrhel6baseimage"]="enm_rhel6_base_image_name"
    ["ERICrhel6jbossimage"]="enm_rhel6_jboss_image_name"
    ["ERICrhel7baseimage"]="enm_rhel7_base_image_name"
    ["ERICrhelpostgresimage"]="vnflaf_db_image"
    ["ERICrhelvnflafimage"]="servicesImage"
    ["httpd_createrepo"]="httpd_createrepo_iso_image"
    ["RHEL_Media"]="rhel6_iso_image"
    ["RHEL_OS_Patch_Set"]="rhel6_updates_iso_image"
)

# Search through the artifacts directory for each file we need.
# Collect the complete filenames for later operation.
# If multiple matches exist, take the first one and pray.
declare -a images
declare -A partials_to_images
echo "*** Searching for all necessary images."
available_files=$(ls "$artifacts")
for file in ${partial_filenames[@]} ; do
    echo "Looking for $file"
    set +e
    echo "$available_files" | grep "^$file"
    if [ $? -ne 0 ] ; then
        echo "Can't find $file !"
        exit 2
    fi
    set -e
    image=$(echo "$available_files" | grep "^$file" | head -1)
    images+=($image)
    partials_to_images[$file]=$image
done
echo "Found all images!"

echo "*** Installing openstack client."
umask 022 && sudo pip -q install python-openstackclient

echo "*** Uploading images to Openstack."
for image in ${images[@]} ; do
    full_path="$artifacts/$image"
    # Parse disk format from the image name.
    disk_format=$(echo "$image" | sed 's/.*\.//g')
    if [ "$disk_format" != "iso" ] && [ "$disk_format" != "qcow2" ] ; then
        echo "Disk format [$disk_format] isn't something I recognize... exiting!"
        exit 3
    fi
    # If an image by this name already exists, and the file size is correct,
    # skip the upload.
    echo "Looking for image: $image"
    set +e
    openstack image show "$image" >/dev/null
    success=$?
    set -e
    if [ "$success" -eq 0 ] ; then
        echo "Image $image already exists in Openstack, testing it."
        # Check the file size in Openstack versus the size on disk.
        openstack_size=$(openstack image show "$image" --column size --format value)
        # Follow any symlinks.. otherwise, stat will return the size of the link itself.
        true_file=$(readlink -m "$full_path")
        disk_size=$(stat -c '%s' "$true_file")
        if [ "$openstack_size" == "$disk_size" ] ; then
            echo "Image in Openstack is the same size as image on disk, continuing."
            continue
        else
            echo "Image in Openstack doesn't seem to match image on disk, deleting it!"
            openstack image delete "$image"
        fi
    fi
    echo "Uploading image: $image"
    openstack image create "$image" --file "$full_path" --disk-format "$disk_format"
done

echo "*** Updating SED file."
# For each image, construct a new line, and replace the preexisting one
# in the SED file.
for partial in ${partial_filenames[@]} ; do
    variable=${variable_names[$partial]}
    image=${partials_to_images[$partial]}
    sed -i "s/$variable\:.*$/$variable\: $image/" "$sed_file"
done

echo "*** Finished!"
