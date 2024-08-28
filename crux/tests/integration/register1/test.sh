#!/bin/sh
abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

# get the linux binaries
cp ../../../build/Linux_x86_64/gobin/registersrv-test .
cp ../../../build/Linux_x86_64/gobin/registercli-test .

echo "Running Register myriadfile"
# Run the myriadfile, starting 1 server, 10 clients with the
# above executables.
# myraid takes nearly 20 seconds to start up 11 docker containers.
# runtime test is limited by registry job  - testsrv.sh has a
# timeout after 15 seconds, In practice registry can clear 10
# clients within 6 seconds of startup.
# All the clients wait until registry service is up before
# they attempt to register.
myriad --config mf.json -t 60s register.mf

rm -f register*-test

# Success test.
# 20 registered reeve/steward keypairs should appear in whitelist.db
RST=`pubkeydb -o whitelist.db/whitelist.db -j | grep -v self | wc -l`
echo "$RST reeve/steward public keys registered"
if [ $RST -eq 20 ]
then
  echo "Success"
  rm -f whitelist.db/whitelist.db
  exit 0
else
  echo "Fail, should be 20 reeve/steward public keys, not $RST"
  exit $RST
fi
