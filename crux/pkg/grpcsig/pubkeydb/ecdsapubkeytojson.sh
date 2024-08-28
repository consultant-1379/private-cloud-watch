#!/bin/bash

if [ $# -le 1 ]
then
	echo 'ecdsa [public key file] AND [service] string must be specified, e.g.: ~/.ssh/id_ecdsa.pub "plogiston-server"'
	echo "usage: ecdsapubkeytojson.sh [public key file] [service]-required [username]-optional"
	exit 1
fi

PUBKEY_PATH=${1%}
SERVICE=${2%}
FINGERPRINT="`ssh-keygen -E md5 -lf ${PUBKEY_PATH}`"

if [ $# == 3 ]
then
	USER_NAME=${3%}
	echo '{"service":"'${SERVICE}'","name":"'${USER_NAME}'","keyid":"/'${SERVICE}'/'${USER_NAME}'/keys/'${FINGERPRINT:8:47}'","pubkey":"'`cat ${PUBKEY_PATH}`'","stateadded":-1}'
	exit 0
fi

echo '{"service":"'${SERVICE}'","name":"'`whoami`'","keyid":"/'${SERVICE}'/'`whoami`'/keys/'${FINGERPRINT:8:47}'","pubkey":"'`cat ${PUBKEY_PATH}`'","stateadded":-1}'

exit 0

