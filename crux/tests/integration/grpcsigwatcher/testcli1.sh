#!/bin/sh
cd /tmp
ssh-add -k .ssh/test-key-rsa
export GRPCSIGCLI_DIAL=grpcsigsrv:50052
export GRPCSIG_FINGERPRINT=e5:6f:35:eb:1b:e9:bb:a8:f1:85:73:39:c5:26:b6:22
export GRPCSIG_USER=maude
sleep 5
echo "===Communicating"
for try in `seq 1 60`
do
	echo "===Call $try"
	sleep 1
	./grpcsigcli
done
echo "===Done"
