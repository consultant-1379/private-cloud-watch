#!/bin/sh
abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

rm -f *-stdio *.crt *.key admincert.tar

myriadca ca-config.json >ca-stdio 2>&1 &
CAPID=$!
sleep 1

myriad --config mf.json -t 400s -v crux.mf
ls -l *-stdio
kill -HUP $CAPID
./get-certs <s0-stdio
go run check-certs.go
go run check-certs-tar.go no-such-file admincert.tar
if [ $? -eq 1 ]
then
	echo "'exit status 1'" is the correct result!
else
	echo "ERROR: expected 'exit status 1'"
	exit 2
fi
