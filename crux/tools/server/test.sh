#!/bin/sh

rm -f *.grpc.go
go build
./server -o . sigtest.proto
diff sigtest.grpc.go sigtest.wanted
rm sigtest.grpc.go
