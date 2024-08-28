#!/bin/bash
#
# This script takes in ENM downloaded media, and outputs a total set of
# artifacts necessary to run ENM on Openstack.
#
# These steps are translated from the rough version of the "ENM on Cloud
# Deployment Instructions" PDF, provided in late October 2017.
#
# None of this should be necessary, since they should be just serving up the
# artifacts that you actually need. Perhaps even as one big tarball. But hey,
# who doesn't love writing shell scripts?
#
# -- Loren, October 2017

# Collect and test necessary arguments.
source_media=$1 ; shift
destination=$1 ; shift

set -eu

if [ -z "$source_media" ] || [ -z "$destination" ] ; then
    echo "Usage: $0 <source media dir> <destination dir>"
    exit 1
fi

if [ ! -d "$source_media" ] ; then
    echo "Source must be a directory containing ENM media!"
    exit 2
fi

# Install dependencies.
echo "*** Installing dependencies."
sudo apt -qy install rpm2cpio genisoimage

# Partial filenames for all the files we expect to see in the source media directory.
enm_iso="ERICenm.*\.iso"
rhel_media="RHEL_Media.*\.iso"
rhel_patch="RHEL_OS_Patch_Set.*\.tar\.gz"
vnflaf="ERICrhelvnflafimage.*\.qcow2"
vnfdb="ERICrhelpostgresimage.*\.qcow2"

# Check to make sure all the files we need are present.
# Ignore any checksum files.
echo "*** Checking to make sure all the files we need are present."
available_files=$(ls "$source_media" | grep -v sha1$ | grep -v md5$ )
test_source () {
    for file in $enm_iso $rhel_media $rhel_patch $vnflaf $vnfdb ; do
        echo "Looking for $file"
        echo "$available_files" | grep "$file" >/dev/null
        found=$?
        if [ $found -ne 0 ] ; then
            echo "Can't find $file !"
            return 1
        fi
    done
}
if ! test_source ; then
    exit 3
fi

# Get the full path to each file we'll be using.
# If there are multiple matches, take the first one, and cross our fingers.
base=$(readlink -m "$source_media")
enm_iso_file="$base"/$(echo "$available_files" | grep "$enm_iso" | head -1)
rhel_media_file="$base"/$(echo "$available_files" | grep "$rhel_media" | head -1)
rhel_patch_file="$base"/$(echo "$available_files" | grep "$rhel_patch" | head -1)
vnflaf_file="$base"/$(echo "$available_files" | grep "$vnflaf" | head -1)
vnfdb_file="$base"/$(echo "$available_files" | grep "$vnfdb" | head -1)

# At this point, we're relatively certain that we can continue, so remove any
# preexisting target directory.
prep_target_dir () {
    # Get the real full path.
    destination=$(readlink -m "$destination")
    here=$(pwd)
    # If the destination is the root or a file or the current directory or the source directory, don't do anything.
    if [ "$destination" == "/" ] ||
        [ -f "$destination" ] ||
        [ "$destination" == "$here" ] ||
        [ "$destination" == "$base" ]  ; then
        echo "I can't use $destination as the target directory."
        return 1
    fi
    # If there's already an existing destination directory, wipe it.
    # This is probably unsafe and unnecessary. Comment it out if you need it.
    #if [ -d "$destination" ]; then
    #    echo "*** Wiping target artifacts directory: $destination"
    #    rm -rf "$destination"
    #fi
    # Recreate it if it's not already there.
    mkdir -p "$destination" || return 1
    echo "*** Testing writes to target directory: $destination"
    touch "$destination"/write_test || return 1
    rm "$destination"/write_test || return 1
}
set +e
prep_target_dir
success=$?
set -e
if [ $success -ne 0 ] ; then
    echo "Can't create files in destination! (error $success)"
    exit $success
fi

# Create links to the original media files from the destination directory.
# You can change this from ln to cp if you need a real copy.
# The patch file will undergo conversion to ISO, so we skip that one.
echo "*** Creating symlinks to the larger media files to avoid making two copies."
for file in "$enm_iso_file" "$rhel_media_file" "$vnflaf_file" "$vnfdb_file" ; do
    ln -sf "$file" "$destination"
done

# Pull rhel6 and rhel7 images out of ENM iso, as well as the "ENM cloud
# templates" RPM.
echo "*** Pulling rhel6, rhel7, and ENM cloud templates RPM from the ENM ISO."
iso_dir="./temp_iso_dir"
mkdir -p "$iso_dir"
sudo mount -o loop "$enm_iso_file" "$iso_dir"
# Create a function, run the function, clean up, check success.
extract_enm () {
    set -e
    cp "$iso_dir"/images/ENM/ERICrhel7baseimage*.qcow2 "$destination" || return 1
    cp "$iso_dir"/images/ENM/ERICrhel6baseimage*.qcow2 "$destination" || return 1
    cp "$iso_dir"/images/ENM/ERICrhel6jbossimage*.qcow2 "$destination" || return 1
    cp "$iso_dir"/cloud/templates/ENM/ERICenmcloudtemplates*.rpm "$destination" || return 1
    cp "$iso_dir"/cloud/templates/ENM/ERICenmdeploymentworkflows*.rpm "$destination" || return 1
    chmod u+rw "$destination"/* || return 1
    return 0
}
set +e
extract_enm
success=$?
set -e
sudo umount "$iso_dir"
rmdir "$iso_dir"
if [ $success -ne 0 ] ; then
    echo "Couldn't copy necessary assets from ENM iso! (error $success)"
    exit $success
fi

# Extract httpd_createrepo.iso and the vnf laf heat templates from the ENM
# cloud templates RPM.
echo "*** Extracting httpd_createrepo.iso and some heat templates from the cloud templates RPM."
cloudtemplates_file="$destination"/$(ls "$destination" | grep "ERICenmcloudtemplates.*\.rpm")
rpm_dir=$(echo "$cloudtemplates_file" | sed 's/.rpm$//')
mkdir -p "$rpm_dir"
# Create a function, run the function, clean up, check success.
extract_cloudtemplates () {
    pushd "$rpm_dir"
    rpm2cpio "$cloudtemplates_file" | cpio -idm
    success=$?
    popd
    if [ $success -ne 0 ] ; then
        echo "Can't extract cloudtemplates rpm!"
        return 1
    fi
    mv "$rpm_dir"/opt/ericsson/ERICenmcloudtemplates*/httpd_createrepo.iso "$destination" || return 1
    for dir in merged_applications merged_volumes infrastructure_resources ; do
        # If the destination directory already exists, remove it.
        sudo rm -rf "$destination"/$dir || return 1
        mv "$rpm_dir"/opt/ericsson/ERICenmcloudtemplates*/$dir "$destination" || return 1
        chmod a+x "$destination"/$dir || return 1
    done
    return 0
}
set +e
extract_cloudtemplates
success=$?
set -e
# We don't need the RPM around anymore; we got everything we need out of it.
rm -rf "$cloudtemplates_file" "$rpm_dir"
if [ $success -ne 0 ] ; then
    echo "Couldn't copy assets from cloudtemplates RPM! (error $success)"
    exit $success
fi

# Convert the RHEL OS patch set from tar.gz to ISO format.
echo "*** Converting the RHEL OS patch set from tar.gz to ISO format."
rhel_dir="rhel_os_patch_set"
mkdir -p "$rhel_dir"
# Create a function, run the function, clean up, check success.
create_rhel_patch_iso () {
    tar -xzf "$rhel_patch_file" -C "$rhel_dir" || return 1
    iso_file_name=$(basename "$rhel_patch_file" | sed 's/.tar.gz$/.iso/')
    genisoimage -R -J -hfs -quiet -o "$destination/$iso_file_name" "$rhel_dir" || return 1
    return 0
}
set +e
create_rhel_patch_iso
success=$?
set -e
rm -rf "$rhel_dir"
if [ $success -ne 0 ] ; then
    echo "Couldn't create RHEL patch iso from tar.gz file! (error $success)"
    exit $success
fi

echo "*** Finished!"
ls -l "$destination"
