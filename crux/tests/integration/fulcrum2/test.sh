#!/bin/sh

abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

case "${MAKECONTAINER:-1}" in
0)	;;
*)	(cd ../../..; make container)
	;;
esac

FLOCKKEY=27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf
BEACON=f99:29718
BFARGS="--beacon $BEACON --key $FLOCKKEY"

case "${WAIT:-0}" in
1)	BFARGS="$BFARGS --wait"
	;;
esac

rm -f *-stdio
(
	echo 'version = "v0.2"'
	for i in `seq 1 ${1:-10}`
	do
		cat <<EOF
		job "f$i" {
			command = "ssh-agent sh -c 'cd /tmp; sleep 5; fulcrum flock $BFARGS --fire 8090'"
		}
EOF
	done
	cat <<EOF
	job "f99" {
		command = "sh -c 'cd /tmp; ip addr ls; fulcrum watch $BFARGS --fire -8090 --n 10'"
	        input "/kv.json" {
	            src = "crux.mf"
	        }
		wait = true
	}
EOF
) > crux.mf
echo "executing myriad"
myriad --config ${CONFIG:-mf.json} -v -t "${TIMEOUT:-180s}" crux.mf
ls -l *-stdio
tail -2 f99-stdio
