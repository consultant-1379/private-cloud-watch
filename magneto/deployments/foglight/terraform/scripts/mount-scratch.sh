#!/bin/bash
#
# Try to identify the block volume that was attached to be used as scratch.
# No great way to figure out which one it actually is, so I just choose the
# first unpartitioned virtual disk.

# mkfs the first volume you find that doesn't seem to be partitioned.
for disk in /dev/vd* ; do
    # Does a partition exist?
    set +e
    lsblk -n "$disk" | grep part >/dev/null
    partition_exists=$?
    set -e

    # Create a new partition and filesystem, if a partition doesn't exist,
    # then exit the loop.
    if [ $partition_exists -ne 0 ] ; then
        echo "Creating filesystem on $disk for scratch."
        echo ";" | sudo sfdisk $disk
        echo "Creating ext4 filesystem in ${disk}1."
        yes | sudo mkfs.ext4 -L scratch "${disk}1"
        break
    fi
done

# Perform any /scratch mounting tasks that don't seem to already be performed,
# including creating the mount point, editing fstab, and mounting. In order to
# figure out which actions to perform, run a group of tests.

# Failures are ok.
set +e
# Does /scratch exist?
ls /scratch >/dev/null
scratch_exists=$?
# Is there a line in /etc/fstab for /scratch ?
grep \/scratch /etc/fstab >/dev/null
fstab_edited=$?
# Is a partition mounted at /scratch? (Doesn't have to be this disk.)
lsblk -n | grep part | grep \/scratch >/dev/null
scratch_mounted=$?
set -e

# Create /scratch, if needed.
if [ $scratch_exists -ne 0 ] ; then
    echo "Creating /scratch mount point."
    sudo mkdir -p /scratch
    sudo chown -R ubuntu:ubuntu /scratch
fi

# Put a line in /etc/fstab for scratch if there isn't already one.
if [ $fstab_edited -ne 0 ] ; then
    echo "Adding /scratch to /etc/fstab."
    echo "LABEL=scratch /scratch    ext4    defaults    0 0" | sudo tee -a /etc/fstab
fi

# Mount, if needed.
if [ $scratch_mounted -ne 0 ] ; then
    echo "Mounting /scratch."
    sudo mount /scratch
fi
