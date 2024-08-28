#!/bin/bash

# Exit with error if any line fails.
set -e

shed="github.com/erixzone/shed"
github_token=$1

echo "Downloading erixzone shed into home directory..."
cd
mkdir -p shed/
cd shed/
git init
git pull https://${github_token}@${shed}
