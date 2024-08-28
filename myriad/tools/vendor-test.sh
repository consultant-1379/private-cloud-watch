#!/bin/bash
# Tests if all dependencies are properly vendored

test=$(basename $0)
incorrect=$(govendor list +external +missing)
if [[ -z "$incorrect" ]]; then
    exit 0
else
    echo "Failed $test: some deps are not properly vendored"
    echo "$ govendor list +external +missing"
    echo $incorrect
    echo "Running 'govendor add +external +missing' may do what you need."
    exit 1
fi
