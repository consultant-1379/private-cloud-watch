#!/bin/bash
#
# Install s3cmd locally and write the .s3cfg file so we don't have to
# provide the default arguments every time we run s3.

access_key=$1 ; shift
secret_key=$1 ; shift
s3_endpoint=$1 ; shift
s3_host="https://$s3_endpoint/"
s3_host_bucket="%(bucket)s.$s3_endpoint"

echo "Installing s3cmd"
sudo apt -qy update || exit $?
sudo apt -qy install python-pip || exit $?
sudo pip install s3cmd || exit $?

echo "Installing root s3cmd configuration"
s3cmd_line="s3cmd --access_key=$access_key --secret_key=$secret_key \
    --host=$s3_host --host-bucket=$s3_host_bucket"
$s3cmd_line --dump-config >~/.s3cfg || exit $?
chmod 600 ~/.s3cfg || exit $?
