#!/bin/bash

export RemoteStore="/tmp/remote-blobstore"
export LocalStore="/tmp/local-blobstore"

mkdir $RemoteStore

mkdir $LocalStore
rm  $RemoteStore/*
rm  $LocalStore/*

Rsrv="127.0.0.1"
Rport="50052"
Lsrv="127.0.0.1"
Lport="50051"

# FIXME:  clear out existing pastiche-testservers first
echo "existing pastiche processes:"
pgrep pastiche

echo "======= Starting Local SERVER  ======="
# Including one non-existent server to test missing server logic
$GOPATH/bin/pastiche-testserver -port=$Lport -servers=$Rsrv:$Rport,127.0.0.1:5005 > local-server.log  2>&1 &
s1pid=$!
echo "Server 1 PID: $s1pid"

echo "======= Starting \"remote\" SERVER   ======="
$GOPATH/bin/pastiche-testserver -port=$Rport  > remote-server.log  2>&1 &
s2pid=$!
echo "Server 2 PID: $s2pid"

sleep 1  # wait for servers to be online
echo "======= Starting DEMO CLIENT ======= with -server_addr=$Lsrv:$Lport -remote_addr=$Rsrv:$Rport"
$GOPATH/bin/pastiche-demo  -server_addr=$Lsrv:${Lport} -remote_addr=$Rsrv:${Rport}  &

read -p "Press return to kill servers"
echo ""
echo "=== Killing started servers"
kill $s1pid $s2pid
