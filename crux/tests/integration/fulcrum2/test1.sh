#!/bin/sh

# fulcrum tomb $BFARGS --debug true --listen $TOMBPORT --fire 8090 --ifile /tmp/ifile --ofile /tmp/ofile
# the important flags for tomb are --ifile INPUT and --ofile OUTPUT.
# INPUT is where a previous output (or the output of synth) is stored.
# OUTPUT is where the current output being generated is stored. (the argument here is tricky;
# i've not figured out why, but it is stored as the basename of OUTPUT in the current directory)

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
TOMBPORT=35167
TOMB=f98:$TOMBPORT
BFARGS="--beacon $BEACON --key $FLOCKKEY"
FILE=tempf

rm -f *-stdio
> $FILE
(
	echo 'version = "v0.2"'
	for i in `seq 1 ${1:-10}`
	do
		cat <<EOF
		job "f$i" {
			command = "ssh-agent sh -c 'cd /tmp; sleep 5; fulcrum flock $BFARGS --fire 8090 # --wait'"
		}
EOF
	done
	cat <<EOF
	job "f98" {
		command = "sh -c 'fulcrum tomb $BFARGS --debug true --listen $TOMBPORT --fire 8090 --ifile /tmp/ifile --ofile /tmp/ofile'"
	        output "/tmp/ofile" {
	            src = "xxhuh"	# this value is not used
	        }
	        input "/tmp/ifile" {
	            src = "m1.out"
	        }
	}
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
myriad --config mf.json -v -t "${TIMEOUT:-180s}" crux.mf
ls -l *-stdio
tail -2 f99-stdio
