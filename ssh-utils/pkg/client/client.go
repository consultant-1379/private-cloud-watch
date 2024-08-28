package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

type SshService int

const (
	SshSvcNone SshService = iota
	SshSvcConnect
	SshSvcForward
)

type SshSvcCfg struct {
	InitFlag bool
	Service  SshService
	DialStr  string
	Cmd      string
}

func RecvSshSvcCfg(data []byte) (*SshSvcCfg, error) {
	cfg := new(SshSvcCfg)
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return cfg, dec.Decode(cfg)
}

func (cfg *SshSvcCfg) Bytes() []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(cfg); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type SshConn struct {
	net.UnixConn
	isTerminal bool
	termstate  *terminal.State
}

func Connect(socketName string, stdio ...int) (*SshConn, error) {
	return SshConnect(socketName, &SshSvcCfg{false, SshSvcConnect, "", ""}, stdio...)
}

func Forward(socketName, dialStr string) (*SshConn, error) {
	return SshConnect(socketName, &SshSvcCfg{false, SshSvcForward, dialStr, ""})
}

func SshConnect(socketName string, cfg *SshSvcCfg, stdio ...int) (*SshConn, error) {
	socketDir := fmt.Sprintf("/var/run/user/%d/transshport", os.Getuid())
	socketPath := fmt.Sprintf("%s/%s", socketDir, socketName)
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{socketPath, "unix"})
	if err != nil {
		return nil, err
	}
	ssh := new(SshConn)
	ssh.UnixConn = *conn
	if len(stdio) < 1 {
		stdio = append(stdio, int(os.Stdin.Fd()))
	}
	if len(stdio) < 2 {
		stdio = append(stdio, int(os.Stdout.Fd()))
	}
	if len(stdio) < 3 {
		stdio = append(stdio, int(os.Stderr.Fd()))
	}
	fds := syscall.UnixRights(stdio...)
	_, _, err = ssh.WriteMsgUnix(cfg.Bytes(), fds, nil)
	if err != nil {
		ssh.Close()
		return nil, err
	}
	return ssh, nil
}

func (ssh *SshConn) SttyRaw() {
	var err error
	ssh.isTerminal = terminal.IsTerminal(int(os.Stdin.Fd()))
	if ssh.isTerminal {
		fmt.Printf("## session open...\n")
		ssh.termstate, err = terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Printf("## stdin terminal still cooked: %s\n", err)
			ssh.termstate = nil
		}
	}
}

func (ssh *SshConn) CloseWait() {
	buf := make([]byte, 4)
	n, err := ssh.Read(buf)
	if ssh.termstate != nil {
		terminal.Restore(int(os.Stdin.Fd()), ssh.termstate)
	}
	if ssh.isTerminal {
		fmt.Printf("## session closed.\n")
	}
	if n != 0 || err != io.EOF {
		fmt.Fprintf(os.Stderr, "## read returned %d %s\n", n, err)
	}
}
