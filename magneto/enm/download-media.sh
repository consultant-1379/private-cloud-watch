#!/bin/bash

media_file=$1 ; shift
destination=$1 ; shift

set -eu

if [ -z "$media_file" ] || [ -z "$destination" ] ; then
    echo "Usage: $0 <media file> <destination dir>"
    exit 1
fi

if [ ! -r "$media_file" ] ; then
    echo "Can't read media file!"
    exit 2
fi

if [ ! -d "$destination" ] ; then
    mkdir -p "$destination"
fi

echo "Warning: you have to be on Ericsson VPN for this to work."
echo "Starting download in 3 seconds..."
sleep 3

wd="$(pwd)"
cd "$destination"
cat "$wd/$media_file" | sed 's/#.*$//' | sed '/^\s*$/d' | while read url ; do
    wget --continue "$url"
done

echo "Complete!"
