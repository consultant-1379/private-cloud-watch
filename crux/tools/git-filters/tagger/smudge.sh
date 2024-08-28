#!/usr/bin/env bash
keyword="GIT_TAG:"
tag=$(git describe --tag HEAD)
while read line
do
    echo $line | sed "s/=$keyword=/=${keyword}$tag=/"
done
