#!/bin/bash
#
# Run apt update & upgrade.

export DEBIAN_FRONTEND=noninteractive

# We have to try this multiple times because digitalocean sucks and sometimes
# running "apt update" on a droplet gives you bunk results.
# Just do it until it succeeds.
#
# Warning: This might put you in an infinite fail loop. I haven't run
# into this problem yet, but I suppose it would be good to try it only a
# certain number of times.
success=1
while [ $success -ne 0 ]; do
    sudo apt -qy update && \
        sudo apt -qy upgrade \
            -o Dpkg::Options::="--force-confdef" \
            -o Dpkg::Options::="--force-confold"
    success=$?
done
