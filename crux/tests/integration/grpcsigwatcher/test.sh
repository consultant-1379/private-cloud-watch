#!/bin/sh
abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

# get the public keys databases
cp ../../../pkg/grpcsig/test/server/pubkeys_test.db .
cp ../../../pkg/grpcsig/test/testdata/pubkeys_test2.db .

# get the matching test secrets for signing
cp ../../../pkg/grpcsig/test/testdata/test-key-rsa .
cp ../../../pkg/grpcsig/test/testdata/test-key-rsa.pub .
cp ../../../pkg/grpcsig/test/testdata/test-key-ecdsa .
cp ../../../pkg/grpcsig/test/testdata/test-key-ecdsa.pub .
cp ../../../pkg/grpcsig/test/testdata/test-key-ed25519 .
cp ../../../pkg/grpcsig/test/testdata/test-key-ed25519.pub .

# get the linux binaries
cp ../../../build/Linux_x86_64/gobin/grpcsigsrv-test .
cp ../../../build/Linux_x86_64/gobin/grpcsigcli-test .


echo "running grpcsig myriadfile"
# Run the myriadfile, starting 1 server, 3 clients with the
# above executables, keys and databases "hot-loaded" into
# the same docker image. Hit server with all 3 clients,
# at 1 request/sec each, and change the whitelist
# database on the fly.
# myriad --config mf.json -d -t 95s grpcsig.mf
myriad --config mf.json -t 95s grpcsig.mf

rm -f pubkeys_*.db
rm -f test-key-*
rm -f grpcsig*-test

# four restarts should be triggered on the server
RST=`cat grpcsigsrv-stdio | grep "whitelist DB restarted pubkeys_test.db" | wc -l`
if [ $RST -eq 4 ]
then
  echo "Success"
  exit 0
else
  exit $RST
fi
