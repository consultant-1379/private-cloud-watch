package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path"
	"plugin"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
	"github.com/erixzone/xaas/platform/ssh-utils/pkg/fdio"
)

func usage() {
	fmt.Printf("usage: %s user@host[:port] [plugin]\n", os.Args[0])
	os.Exit(127)
}

var needsInit bool

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		usage()
	}
	args := strings.SplitN(os.Args[1], "@", 2)
	if len(args) != 2 {
		usage()
	}
	username := args[0]
	hostname := args[1]
	dialstr := hostname
	if !strings.Contains(dialstr, ":") {
		dialstr += ":22"
	}
	needsInit = (len(os.Args) > 2)

	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Who do you think you are: %s", err)
	}
	hkeyCallback, err := knownhosts.New(path.Join(usr.HomeDir, ".ssh/known_hosts"))
	if err != nil {
		log.Fatalf("knownhosts.New failed: %s", err)
	}

	fmt.Print("Enter password: ")
	pwd, err := terminal.ReadPassword(syscall.Stdin)
	fmt.Print("\n")
	if err != nil {
		log.Fatalf("ReadPassword failed: %s", err)
	}

	socketDir := fmt.Sprintf("/var/run/user/%d/transshport", os.Getuid())
	err = os.MkdirAll(socketDir, 0700)
	if err != nil {
		log.Fatalf("MkdirAll(\"%s\") failed: %s", socketDir, err)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback: hkeyCallback,
	}
	c, err := ssh.Dial("tcp", dialstr, config)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer c.Close()

	socketPath := fmt.Sprintf("%s/%s", socketDir, hostname)
	os.Remove(socketPath) // ignore error
	defer os.Remove(socketPath)
	ou := syscall.Umask(0177)
	ln, err := net.ListenUnix("unix", &net.UnixAddr{socketPath, "unix"})
	syscall.Umask(ou)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %s", socketPath, err)
	}
	log.Printf("## PID %d listening on %s ...", os.Getpid(), socketPath)
	go func() {
		connCount := 0
		for {
			conn, err := ln.AcceptUnix()
			if err != nil {
				log.Printf("unAcceptable on %s: %s", socketPath, err)
				break
			}
			connCount++
			log.Printf("# %d # connection accepted", connCount)
			go newRemote(c, connCount, conn)
		}
	}()
	if needsInit {
		go doInit(os.Args[2], hostname)
	}
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Signal(syscall.SIGINT))
	sig := <-exitChan
	log.Printf("## Exiting on %s.", sig)
}

func newRemote(c *ssh.Client, id int, conn *net.UnixConn) {
	defer conn.Close()
	cfgbuf := make([]byte, 32768)
	fdbuf := make([]byte, syscall.CmsgSpace(3*4)) // 3 fd's
	cfglen, _, _, _, err := conn.ReadMsgUnix(cfgbuf, fdbuf)
	if err != nil {
		log.Printf("# %d # Recvmsg failed: %s", id, err)
		return
	}
	msgs, err := syscall.ParseSocketControlMessage(fdbuf)
	if err != nil {
		log.Printf("# %d # ParseSocketControlMessage failed: %s", id, err)
		return
	}
	var fds []int
	for _, msg := range msgs {
		msgfds, err := syscall.ParseUnixRights(&msg)
		if err != nil {
			log.Printf("# %d # ParseUnixRights failed: %s", id, err)
			return
		}
		for _, fd := range msgfds {
			defer syscall.Close(fd)
			fds = append(fds, fd)
		}
	}
	if len(fds) != 3 {
		log.Printf("# %d # newRemote: got %d fd's, expected %d", id, len(fds), 3)
		return
	}
	cfg, err := client.RecvSshSvcCfg(cfgbuf[:cfglen])
	if err != nil {
		log.Printf("# %d # RecvSshSvcCfg: %s", id, err)
		return
	}
	if needsInit && !cfg.InitFlag {
		log.Printf("# %d # waiting for init", id)
		return
	}
	if cfg.Service == client.SshSvcConnect {
		newSession(c, id, fds, cfg.Cmd)
	} else {
		newForward(c, id, fds, cfg.DialStr)
	}
}

func newSession(c *ssh.Client, id int, fds []int, cmd string) {
	session, err := c.NewSession()
	if err != nil {
		log.Printf("# %d # Failed to create session: ", id, err)
		return
	}
	defer session.Close()

	src, err := fdio.NewHupReader(fds[0])
	if err != nil {
		log.Printf("# %d # NewHupReader: ", id, err)
		return
	}
	defer src.Hangup()
	session.Stdin = src
	session.Stdout = fdio.FdWriter(fds[1])
	session.Stderr = fdio.FdWriter(fds[2])
	switch {
	case cmd != "":
		log.Printf("# %d # Cmd starting...", id)
		err = session.Run(cmd)
	default:
		log.Printf("# %d # Shell starting...", id)
		err = doPty(session, id, src.Fd())
		if err == nil {
			err = session.Shell()
		}
	}
	if err != nil {
		log.Printf("# %d # %s", id, err)
		return
	}
	err = session.Wait()
	log.Printf("# %d # connection closed, err = %v", id, err)
}

func doPty(session *ssh.Session, id, stdin int) error {
	width, height := 160, 40
	if terminal.IsTerminal(stdin) {
		var err error
		width, height, err = terminal.GetSize(stdin)
		if err != nil {
			log.Printf("# %d # terminal.GetSize failed: ", id, err)
			width, height = 160, 40
		}
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 4915200,
		ssh.TTY_OP_OSPEED: 4915200,
	}
	if err := session.RequestPty("xterm", height, width, modes); err != nil {
		return fmt.Errorf("RequestPty failed: ", err)
	}
	return nil
}

func newForward(c *ssh.Client, id int, fds []int, dialstr string) {
	rconn, err := c.Dial("tcp", dialstr)
	if err != nil {
		log.Printf("# %d # client.Dial failed: %s", id, err)
		return
	}
	defer rconn.Close()
	log.Printf("# %d # client.Dial to %s", id, dialstr)
	closed := make(chan string, 2)
	go func() {
		log.Printf("# %d # read remote plumbed", id)
		io.Copy(fdio.FdWriter(fds[1]), rconn)
		log.Printf("# %d # read remote EoF", id)
		closed <- "remote"
	}()
	go func() {
		log.Printf("# %d # read local plumbed", id)
		io.Copy(rconn, fdio.NonZeroReader{fdio.FdReader(fds[0])})
		log.Printf("# %d # read local EoF", id)
		closed <- "local"
	}()
	sfd := <-closed
	log.Printf("# %d # connection closed after read %s EoF", id, sfd)
}

func doInit(plugName, hostname string) {
	if !strings.Contains(plugName, "/") {
		plugName = "ssh-" + plugName + "-plugin"
		plugName = path.Join(os.Getenv("GOPATH"), "bin", plugName)
	}
	plug, err := plugin.Open(plugName)
	if err != nil {
		log.Printf("## doInit Open: %s", err)
		os.Exit(1)
	}
	initer, err := plug.Lookup("Main")
	if err != nil {
		log.Printf("## doInit Lookup: %s", err)
		os.Exit(1)
	}
	log.Printf("## doInit starting...")
	err = initer.(func(string)error)(hostname)
	if err != nil {
		log.Printf("## doInit failed: %s", err)
		os.Exit(1)
	}
	log.Printf("## doInit succeeded")
	needsInit = false
}
