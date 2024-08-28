#!/bin/bash

set -e

sudo apt -qy update
sudo apt -qy install nginx
sudo service nginx start
