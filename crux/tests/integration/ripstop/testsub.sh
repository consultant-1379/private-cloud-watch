#!/usr/bin/env bash

if [ $# != 2 ]
then
        echo "usage: `basename $0` [0 - no compile, 1 -compile containers] [20 - even total number of jobs]"
        exit 1
fi
echo Compile: $1
echo Number of Jobs: $2

JOB=$(($2))
expectedPST=$JOB
expectedPST=0 # no longer testing pastiche in ripstop
EPS=$((JOB+expectedPST+1))
SHK=$((JOB/2))
echo shark horde: 1 - $SHK containers
JET=$((SHK+1))
echo jet horde: $JET - $JOB containers

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

rm -f *-stdio
(
	echo 'version = "v0.2"'
	cat <<EOF
	job "f99" {
		command = "sh -c 'cd /tmp; ripstop watch $BFARGS --n 1'"
		wait = true
	}
EOF
	for (( i = 1; i <= ${SHK}; i++ ))
	do
	cat <<EOF
	job "f$i" {
		command = "ssh-agent sh -c 'cd /tmp; sleep 16; ripstop flock $BFARGS'"
		wait = true
	}
EOF
	done
	for (( j = ${JET}; j <= ${JOB}; j++ ))
	do
	cat <<EOF
	job "f$j" {
		command = "ssh-agent sh -c 'cd /tmp; sleep 15; ripstop flock $BFARGS --horde jets; ls -l .muck'"
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
time myriad --config mf.json -t 300s crux.mf

#Some info
echo "myriad finished"
ls -l *-stdio
echo "f99 Summary:"
grep flocks f99-stdio
echo "All should reach - completed!!"
grep "Ripstop completed" *-stdio

OK=0
# Success test part 1
# 10 Reeve Servers started
DST=`grep "Serving /Reeve"  *-stdio | wc -l`
if [ $DST -eq $JOB ]
then
  echo "Success - $JOB Reeve Servers started"
else
  echo "Fail - should be $JOB Reeve Servers started, not $DST"
  OK=1
fi

# Success test part 2
# 1 Registry Server started
CST=`grep "Serving /Registry" *-stdio | wc -l`
if [ $CST -eq 1 ]
then
  echo "Success - 1 Registry Server started"
else
  echo "Fail - should be 1 Registry Server started, not $CST"
  OK=2
fi

# Success test part 3
# 1 Steward Server started
BST=`grep "Serving /Steward" *-stdio | wc -l`
if [ $BST -eq 1 ]
then
  echo "Success - 1 Steward Server started"
else
  echo "Fail - should be 1 Steward Servers started, not $BST"
  OK=3
fi

# Success test part 4
# 10 registered reeves should appear in *-stdio
RST=`grep REGISTERED *-stdio | wc -l`
if [ $RST -eq $JOB ]
then
  echo "Success - $JOB Reeves Registered"
else
  echo "Fail - should be $JOB Registered Reeves, not $RST"
  OK=4
fi

# Success test part 5
# 10 Reeves can communicate with Steward
SST=`grep "This reeve can communi" *-stdio | wc -l`
if [ $SST -eq $JOB ]
then
  echo "Success - $JOB Reeve Servers communicating with Steward"
else
  echo "Fail - should be $JOB Reeve Servers communicating with Steward, not $SST"
  OK=5
fi

# Success test part 6
# 10 Pastiche Servers started
PST=`grep Serving *-stdio | grep pastiche | grep -v info | wc -l`
if [ $PST -eq $expectedPST ]
then
  echo "Success - $expectedPST Pastiche Servers started"
else
  echo "Fail - should be $expectedPST Pastiche Servers started, not $PST"
  OK=6
fi

# Success test part 7
# 20 Endpoints sent to Steward for registration
AST=`grep "Steward processing endpoint" *-stdio | wc -l`
if [ $AST -eq $EPS ]
then
  echo "Success - $EPS Endpoints Sent to Steward"
else
  echo "Fail - should be $EPS Endpoints Sent to Steward, not $AST"
  OK=7
fi

# Success test part 8
# 20 Clients sent to Steward for registration
EST=`grep "Steward processing client" *-stdio | wc -l`
if [ $EST -eq $EPS ]
then
  echo "Success - $EPS Clients Sent to Steward"
else
  echo "Fail - should be $EPS Clients Sent to Steward, not $EST"
  OK=8
fi

# Success test part 9
# 10 EndpointsUp results returned
EPU=`grep "EndpointsUp result:" *-stdio | wc -l`
if [ $EPU -eq $expectedPST ]
then
  echo "Success - $EPU EndpointsUp requests returned for Pastiche"
else
  echo "Fail - should be $expectedPST EndpointsUp requests returned for Pastiche, not $EPU"
  OK=9
fi

# Success test part 10
# 10 Catalog results returned
CAT=`grep "Catalog result:" *-stdio | wc -l`
if [ $CAT -eq $expectedPST ]
then
  echo "Success - $CAT Catalog requests returned for Pastiche"
else
  echo "Fail - should be $expectedPST Catalog requests returned for Pastiche, not $CAT"
  OK=10
fi

exit $OK
