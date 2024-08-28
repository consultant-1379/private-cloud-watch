#!/bin/bash

# Usage: ./build-deps.sh github.com/some/go/project <SHA1>
usage="$0 github.com/some/go/project <SHA1>"
project=$1
sha1=$2

if [ -z "$project" -o -z "$sha1" ]; then
    echo $usage
    exit 1
fi

if [[ ! -d "$GOPATH/src/$project" ]]; then
    go get -u $project
fi

cd $GOPATH/src/$project
git checkout -fq $sha1 2>&1 > /dev/null
rtv=$?

# If the checkout failed, may just be out of date.
# Try fetching from remote and checking out again.
if [[ $rtv -ne 0 ]]; then
    git fetch -fq
    git checkout -fq $sha1
    rtv=$?
    # Give up if we're still failing.
    if [[ $rtv -ne 0 ]]; then
        exit $rtv
    fi
fi

go install $project
exit $?
