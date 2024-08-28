package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
	"github.com/erixzone/xaas/platform/ssh-utils/pkg/fdio"
)

func _main() int {
	if len(os.Args) != 4 {
		fmt.Printf("usage: %s port sockname dialstr\n", os.Args[0])
		return 127
	}
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+os.Args[1])
	if err != nil {
		fmt.Printf("net.ResolveTCPAddr failed: %s\n", err)
		return 1
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Printf("net.ListenTCP failed: %s\n", err)
		return 1
	}
	var connCount int
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("unAcceptable on %s: %s", os.Args[1], err)
			break
		}
		connCount++
		log.Printf("# %d # connection accepted", connCount)
		go newConn(connCount, conn)
	}
	return 0
}

func newConn(id int, conn *net.TCPConn) {
	defer conn.Close()
	var err error
	var stdin, stdout [2]int
	err = syscall.Pipe(stdin[:])
	if err == nil {
		err = syscall.Pipe(stdout[:])
	}
	if err != nil {
		log.Printf("# %d # Pipe failed: %s\n", id, err)
		return
	}
	rconn, err := client.SshConnect(os.Args[2], &client.SshSvcCfg{false, client.SshSvcForward, os.Args[3], ""}, stdin[0], stdout[1])
	syscall.Close(stdin[0])
	syscall.Close(stdout[1])
	defer syscall.Close(stdin[1])
	defer syscall.Close(stdout[0])
	if err != nil {
		log.Printf("# %d # Forward %s %s failed: %s\n", id, os.Args[2], os.Args[3], err)
		return
	}
	defer rconn.Close()
	closed := make(chan string, 2)
	go func() {
		log.Printf("# %d # read remote plumbed", id)
		io.Copy(fdio.FdWriter(stdin[1]), conn)
		log.Printf("# %d # read remote EoF", id)
		closed <- "remote"
	}()
	go func() {
		log.Printf("# %d # read local plumbed", id)
		io.Copy(conn, fdio.NonZeroReader{fdio.FdReader(stdout[0])})
		log.Printf("# %d # read local EoF", id)
		closed <- "local"
	}()
	sfd := <-closed
	log.Printf("# %d # connection closed after read %s EoF", id, sfd)
}

func main() {
	os.Exit(_main())
}
