#!/bin/bash

if [ $# != 2 ]
then
	echo "usage: `basename $0` [number of tests] [nodes per test]"
	exit 1
fi
echo "number of tests: $1"
echo "nodes per test: $2"

abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
cd $DIR

# compile once
(cd ../../..; make container)

rm -fr multipass multifail
mkdir multipass multifail

NUMCYC=0
while [ $NUMCYC != $1 ]
do
	let NUMCYC=$NUMCYC+1

	RUNNUM=`printf %07d $NUMCYC`

	# Do run
	echo Myriad Run: $NUMCYC of $1 $RUNNUM
	echo  "./testsub.sh 0 $2 > $RUNNUM.out"
	./testsub.sh 0 $2 > $RUNNUM.out
	if [ $? -eq 0 ]; then
		echo Run OK
		mv $RUNNUM.out multipass/
		# Not keeping the *-stdio here
	else
		echo Run FAILED
		# Keep everything
		mkdir multifail/$RUNNUM
		mv $RUNNUM.out multifail/$RUNNUM/
		mv *-stdio multifail/$RUNNUM/
	fi

done
echo Done multi-run test!
exit 0
