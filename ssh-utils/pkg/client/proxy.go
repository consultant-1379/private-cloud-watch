package client

import (
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/fdio"
)

type proxyAddr string

func (a *proxyAddr) Network() string {
	return "ssh-proxy"
}

func (a *proxyAddr) String() string {
	return string(*a)
}

type ProxyConn struct {
	SshConn
	srcAddr  proxyAddr
	destAddr proxyAddr
	src      io.ReadCloser
	dest     io.WriteCloser
}

func ProxyDial(socketName, dialStr string) (*ProxyConn, error) {
	var err error
	var stdin, stdout [2]int
	err = syscall.Pipe(stdin[:])
	if err == nil {
		err = syscall.Pipe(stdout[:])
	}
	if err != nil {
		return nil, err
	}
	conn, err := SshConnect(socketName, &SshSvcCfg{false, SshSvcForward, dialStr, ""}, stdin[0], stdout[1])
	syscall.Close(stdin[0])
	syscall.Close(stdout[1])
	if err != nil {
		syscall.Close(stdin[1])
		syscall.Close(stdout[0])
		return nil, fmt.Errorf("Connect %s failed: %s\n", socketName, err)
	}
	r := new(ProxyConn)
	r.SshConn = *conn
	r.srcAddr = proxyAddr(socketName)
	r.destAddr = proxyAddr(dialStr)
	r.src = fdio.NonZeroReader{fdio.FdReader(stdout[0])}
	r.dest = fdio.FdWriter(stdin[1])
	return r, nil
}

func SshClient(conn net.Conn, config *ssh.ClientConfig) (*ssh.Client, error) {
	c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func (r *ProxyConn) SshClient(config *ssh.ClientConfig) (*ssh.Client, error) {
	return SshClient(r, config)
}

func (r *ProxyConn) Read(b []byte) (int, error) {
	return r.src.Read(b)
}

func (r *ProxyConn) Write(b []byte) (int, error) {
	return r.dest.Write(b)
}

func (r *ProxyConn) Close() error {
	r.src.Close()
	r.dest.Close()
	return r.SshConn.Close()
}

func (r *ProxyConn) LocalAddr() net.Addr {
	return &r.srcAddr
}

func (r *ProxyConn) RemoteAddr() net.Addr {
	return &r.destAddr
}

func (r *ProxyConn) SetDeadline(t time.Time) error {
	return nil
}

func (r *ProxyConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (r *ProxyConn) SetWriteDeadline(t time.Time) error {
	return nil
}
