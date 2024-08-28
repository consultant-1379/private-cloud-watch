#!/bin/bash

set -e

# Check for local dependencies.
dependencies="pip curl wget"
for dep in $dependencies ; do
    if ! which $dep >/dev/null ; then
        echo "Can't find '$dep' executable in path, can't continue."
        return 2
    fi
done

dest_dir="$HOME/.local/bin"
terraform_url='https://releases.hashicorp.com/terraform/0.11.10/terraform_0.11.10_linux_amd64.zip'
kubectl_url='https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/kubectl'
authenticator_url='https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator'

mkdir -p "$dest_dir"
cd "$dest_dir"

wget "$terraform_url" \
    && unzip -o terraform_* \
    && rm terraform_* \
    && chmod 755 terraform

curl -o kubectl "$kubectl_url" \
    && chmod 755 kubectl

curl -o aws-iam-authenticator "$authenticator_url" \
    && chmod 755 aws-iam-authenticator

pip install awscli --upgrade --user

