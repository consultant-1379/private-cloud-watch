#!/bin/bash
#
# Copy the most recent ldap database backup to the locally configured S3 object
# store. The backups are generated automatically by the ldap server container.

set -e

service_name=$1
if [ -z "$service_name" ]; then echo "No service name!" ; exit 1 ; fi

# Copy the 2 most recent files to s3.
backup_dir="/srv/slapd/backup"
ls -t $backup_dir | head -n2 | while read file ; do
    /usr/local/bin/s3cmd put "$backup_dir/$file" s3://$service_name/
done

