#!/bin/bash
#
# Snapshot the consul database running in docker with the given service name,
# and copy the snapshot to S3.

set -e

service_name=$1
if [ -z "$service_name" ]; then echo "No service name!" ; exit 1 ; fi

snapshot="/consul/snapshot/$(date +%Y-%m-%d-%H-%M).snap"

docker exec $(docker ps -qf name=$service_name) consul snapshot save $snapshot

/usr/local/bin/s3cmd put /srv/$snapshot s3://$service_name/
