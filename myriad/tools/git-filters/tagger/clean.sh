#!/bin/bash
keyword="GIT_TAG:"
while read line
do
    echo $line | sed "s/=$keyword[^=]*=/=${keyword}=/"
done
