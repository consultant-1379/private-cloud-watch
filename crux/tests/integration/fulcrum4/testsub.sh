#!/usr/bin/env bash

if [ $# != 2 ]
then
        echo "usage: `basename $0` [0 - no compile, 1 -compile containers] [20 - even total number of jobs]"
        exit 1
fi
echo Compile: $1
echo Number of Jobs: $2

abspath() {
	echo "$(cd "$(dirname "${1}")"; pwd)/$(basename "${1}")"
}

DIR=$(dirname $(abspath $0))
JOB=$(($2))
EPS=$(($2+$2))
SHK=$(($2/2))
echo shark horde: 1 - $SHK containers
JET=$(($SHK+1))
echo jet horde: $JET - $JOB containers

cd $DIR

if [ $1 -eq 1 ]
then
	(cd ../../..; make container)
fi

rm -f *-stdio
rm -fr /tmp/cache
(
	echo 'version = "v0.2"'
	cat <<'EOF'
	job "f99" {
		command = "sh -c 'cd /tmp; fulcrum watch --beacon f99:29718 --key 27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf --n 1'"
		wait = true
	}
EOF
	for (( i = 1; i <= ${SHK}; i++ ))
	do
	cat <<EOF
	job "f$i" {
		command = "ssh-agent sh -c 'mkdir /tmp/cache; muster0 /tmp/cache /crux/bin/plugin_* | sh; cd /tmp; sleep 5; fulcrum flock --beacon f99:29718 --key 27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf --ip f$i --name f$i --horde sharks'"
		wait = true
	}
EOF
	done
	for (( j = ${JET}; j <= ${JOB}; j++ ))
	do
	cat <<EOF
	job "f$j" {
		command = "ssh-agent sh -c 'mkdir /tmp/cache; muster0 /tmp/cache /crux/bin/plugin_* | sh; cd /tmp; cat /tmp/cache/symtab; sleep 5; fulcrum flock --beacon f99:29718 --key 27670d559d41e1d1cb3ce9c71ba5081330e5e5e5853541b89f71a3209b0641bf --ip f$j --name f$j --horde jets'"
		wait = true
	}
EOF
	done
) > crux.mf

echo "executing myriad"
time myriad --config mf.json -t 400s crux.mf

#Some info
echo "myriad finished"
ls -l *-stdio
echo "f99 Summary:"
grep flocks f99-stdio

cat *-stdio | egrep 'ServiceCount|Demo' | sort | uniq -c | awk -v JOBS=$JOB '
function check(svc, n){
	if(count[svc] != n){
		printf("%s: count should be %d but is %d\n", svc, n, count[svc])
		exit(1)
	}
}
/ServiceCount/ {
	if($1 != JOBS) { printf("count should be %d but is %d: %s\n", JOBS, $1, $0); exit(1) }
	svc = $3; sub(".*/", "", svc)
	count[svc] += $4
}
/Demo/ {
	if(match($0, "SUCCESS") > 0) count["SUCCESS"]++
	if(match($0, "FAILURE") > 0) count["FAILURE"]++
}
END {
	check("Picket", JOBS)
	check("Steward", 1)
	check("Reeve", JOBS)
	#check("FAILURE", 1)
	#check("SUCCESS", 6)
	printf("test passed ok\n")
	exit(0)
}'

exit $OK
