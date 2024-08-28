#!/usr/bin/env bash

if [ $# != 2 ]
then
        echo "usage: `basename $0` [0 - no compile, 1 -compile containers] [20 - even total number of jobs]"
        exit 1
fi
echo Compile: $1
echo Number of Jobs: $2

JOB=$(($2))
EPS=$(($2+$2+1))
SHK=$(($2/2))

abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

sizedockernet() {
	# $1 is the number of hosts that must fit on the subnet.
	# /29 is the smallest practical CIDR size.
	# /16 is the docker default, myriad can't go larger
	# without a specific network address.
	for (( cidrsize=29 ; cidrsize >= 16 ; cidrsize-- ))
	do
		n=$(( (1<<(32-cidrsize))-3 )) # net, b'cast, g'way
		[ $((n)) -ge $(($1)) ] && break
	done
	echo $cidrsize
}

DIR=$(dirname $(abspath $0))
cd $DIR

if [ $1 -eq 1 ]
then
	(cd ../../..; make container)
fi

#FLOCKKEY=$(ripstop key)
FLOCKKEY=27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf
BEACON=f99:29718
BFARGS="--beacon $BEACON --key $FLOCKKEY --fire 8090"
HORDES="--hordes sharks:$SHK:jets:"

rm -f *-stdio
(
	echo 'version = "v0.2"'
	cat <<EOF
	job "f99" {
		command = "sh -c 'cd /tmp; organza watch $BFARGS --n 1'"
		wait = true
	}
EOF
	for (( i = 1; i <= ${JOB}; i++ ))
	do
	cat <<EOF
	job "f$i" {
		command = "ssh-agent sh -c 'mkdir /tmp/cache; muster0 /tmp/cache /crux/bin/plugin_* | sh; cd /tmp; sleep 6; organza flock $BFARGS $HORDES'"
		output "/tmp/.muck/steward/steward.db" {
                        dst = "steward-f$i"
                }
                output "/tmp/.muck/whitelist.db" {
                        dst = "whitelist-f$i"
                }
		wait = true
	}
EOF
	done
	) > crux.mf

cidrsize=`sizedockernet $((JOB + 1))` # add one for beacon
echo "cidrsize: $cidrsize"

cat <<EoF >mf.json
{
  "docker.image" : "erixzone/crux-main",
  "docker.network.subnet" : "/$cidrsize",
  "timeout": "4000s",
  "out":"."
}
EoF

echo "executing myriad"
time myriad --config mf.json -t 240s crux.mf

#Some info
echo "myriad finished"
ls -l *-stdio
echo "f99 Summary:"
grep flocks f99-stdio
echo "All should reach - completed!!"
grep "Organza completed" *-stdio
