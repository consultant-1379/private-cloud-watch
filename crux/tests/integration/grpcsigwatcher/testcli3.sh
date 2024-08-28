#!/bin/sh
cd /tmp
ssh-add -k .ssh/test-key-ed25519
export GRPCSIGCLI_DIAL=grpcsigsrv:50052
export GRPCSIG_FINGERPRINT=6f:d4:de:43:e2:6c:de:7c:44:2d:33:4a:d1:35:ab:8a
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
