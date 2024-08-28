#!/bin/bash
#
# Mount a block volume with the given name to the given mountpoint.
# If we find an unpartitioned disk attached, mkfs it and assign the
# given name to it before continuing.
#
# This obviously should only be run in a highly amenable environment;
# otherwise, the results might not be what you want.
#

volname=$1 ; shift
mountpoint=$1 ; shift

if [ -z "$volname" ] || [ -z "$mountpoint" ]; then
    echo "Usage: $0 <volume label> <mount point>"
    exit 1
fi

# Try to identify the block volume that was attached, then mkfs it.
# No great way to figure out which one it actually is, so I just choose the
# first unpartitioned disk.
#
# We could also allow the user to pass the device name on the command line.
# Future revision?
lsblk -ldno NAME | while read disk ; do
    device="/dev/$disk"
    # Does a partition exist? If so, pass over this one.
    lsblk -n "$device" | grep " part " >/dev/null
    if [ $? -eq 0 ] ; then continue ; fi

    # Is it a loop device? If so, ignore it.
    lsblk -n "$device" | grep " loop " >/dev/null
    if [ $? -eq 0 ] ; then continue ; fi

    # Create partition.
    echo "Creating partition on $disk for $volname."
    echo ";" | sudo sfdisk "$device"

    # Identify new partition.
    while [[ ! $partition ]] ; do
        partition=`lsblk -ln "$device" | grep " part " | awk '{print $1}'`
    done

    # Create filesystem.
    echo "Creating ext4 filesystem on /dev/${partition}."
    yes | sudo mkfs.ext4 -L "$volname" "/dev/${partition}"
    # Only do this once.
    break
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
    echo "LABEL=$volname   $mountpoint    ext4    defaults    0 0" | sudo tee -a /etc/fstab
fi

# Mount, if needed.
if [ $mount_mounted -ne 0 ] ; then
    echo "Mounting $mountpoint"
    sudo mount "$mountpoint"
fi
