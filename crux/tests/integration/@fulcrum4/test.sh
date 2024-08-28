#!/bin/sh
abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

(cd ../../..; make container)
rm -f *-stdio
echo "executing myriad"
myriad --config mf.json -t 100s crux.mf
ls -l *-stdio
tail -2 f99-stdio
