#!/usr/bin/env bash

set -e

CONFIG_LOCATION="/usr/local/etc/fmalarm"

echo "### Installing PIP"
python ./wheels/pip-9.0.1-py2.py3-none-any.whl/pip install --no-index --find-links=./wheels pip

echo "### Installing Dependecies"
pip install --no-index --find-links ./wheels -r requirements.txt

echo "### Installing Module"
pip install .

echo "### Setting up config file"
HAPROXY_IP=$(grep int_haproxy_internal /ericsson/tor/data/global.properties | awk -F '=' '{print $2}' | awk -F ',' '{print $1}')
mkdir -p $CONFIG_LOCATION
cp ./alarm.conf.sample $CONFIG_LOCATION/alarm.conf
sed -i -e "s/<HAPROXY_IP>/$HAPROXY_IP/g" $CONFIG_LOCATION/alarm.conf

