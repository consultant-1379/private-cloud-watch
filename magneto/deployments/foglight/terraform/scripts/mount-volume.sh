#!/bin/bash
#
# Mount a block volume with the given name to the given mountpoint.
# If we find an unpartitioned disk attached, mkfs it and assign the
# given name to it before continuing.
#
# This obviously should only be run in a highly controlled environment;
# otherwise, the results might not be what you want.
#

name=$1 ; shift
mountpoint=$1 ; shift

# Try to identify the block volume that was attached, then mkfs it.
# No great way to figure out which one it actually is, so I just choose the
# first unpartitioned disk.
lsblk -ldno NAME | while read disk ; do
    device="/dev/$disk"
    # Does a partition exist?
    lsblk -n "$device" | grep part >/dev/null
    partition_exists=$?
    if [ $partition_exists -ne 0 ] ; then
        echo "Creating filesystem on $disk for $name."
        echo ";" | sudo sfdisk "$device"
        echo "Creating ext4 filesystem at ${device}1."
        yes | sudo mkfs.ext4 -L "$name" "${device}1"
        # Only do this once.
        break
    fi
done

# Perform any mounting tasks that don't seem to already be performed, including
# creating the mount point, editing fstab, and mounting. In order to figure out
# which actions to perform, run a group of tests.

# Does mountpoint exist?
ls "$mountpoint" >/dev/null
mount_exists=$?

# Is there a line in /etc/fstab for the mountpoint?
grep "$mountpoint" /etc/fstab >/dev/null
fstab_edited=$?

# Is a partition mounted at the mountpoint? (Doesn't have to be this disk.)
lsblk -n | grep part | grep "$mountpoint" >/dev/null
mount_mounted=$?
set -e

# Create mountpoint, if needed.
if [ $mount_exists -ne 0 ] ; then
    echo "Creating $mountpoint mount point."
    sudo mkdir -p "$mountpoint"
    sudo chown -R ubuntu:ubuntu "$mountpoint"
fi

# Put a line in /etc/fstab for this mount if there isn't already one.
if [ $fstab_edited -ne 0 ] ; then
    echo "Adding $mountpoint to /etc/fstab."
    echo "LABEL=$name   $mountpoint    ext4    defaults    0 0" | sudo tee -a /etc/fstab
fi

# Mount, if needed.
if [ $mount_mounted -ne 0 ] ; then
    echo "Mounting $mountpoint"
    sudo mount "$mountpoint"
fi
