#!/usr/bin/env bash

set -e
# build and install a cdep bundle from an archive
case $# in
5)	;;
*)	echo "Usage: $0 prefix github_path github_project snapshot archive" 1>&2; exit 1
esac
prefix=$1
github_path=$2
github_project=$3
snapshot=$4
archive=$5

src_path=src/$github_path
project=$src_path/$github_project

cd $prefix

# make destination dirs
mkdir -p ./$github_project/{pkg,bin,$src_path}
# clean up any prior detritus
rm -fr ./$github_project/$project
# unpack and move to where it needs to be
tar zxvf $archive >tar.errors 2>&1 || cat tar.errors
mv $snapshot ./$github_project/$project
# go install it!
cd $github_project/$project
GOPATH=$prefix/$github_project make install
# install it where it needs to go
mkdir -p $GOPATH/bin
list=`make --no-print-directory export-list 2>/dev/null || true`
for i in ${list:-$github_project}
do
	cp $prefix/$github_project/bin/$i $GOPATH/bin/$i
done
# HACK(skaar): work around recursive go build commands picking up builds/$dep deps
rm -fr $prefix/$github_project/{src,pkg}
