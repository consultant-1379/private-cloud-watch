#!/bin/sh

exec >>"$1" 2>&1
shift
exec prometheus "$@"
