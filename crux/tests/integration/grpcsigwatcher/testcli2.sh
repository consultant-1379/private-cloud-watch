#!/bin/sh
cd /tmp
ssh-add -k .ssh/test-key-ecdsa
export GRPCSIGCLI_DIAL=grpcsigsrv:50052
export GRPCSIG_FINGERPRINT=2c:f3:65:41:56:97:6c:2d:aa:08:ef:34:f3:ef:e3:c8
export GRPCSIG_USER=bobo
sleep 5
echo "===Communicating"
for try in `seq 1 60`
do
	echo "===Call $try"
	sleep 1
	./grpcsigcli
done
echo "===Done"
