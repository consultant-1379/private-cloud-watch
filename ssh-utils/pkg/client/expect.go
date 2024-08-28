package client

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/expect"
	"github.com/erixzone/xaas/platform/ssh-utils/pkg/fdio"
)

type ExpecterClient struct {
	SshConn
	expect.Expecter
	src     fdio.TimeoutReader
	dest    fdio.FdWriter
	verbose bool
}

func NewExpecterClient(socketName string, initFlag ...bool) (*ExpecterClient, error) {
	var err error
	var stdin, stdout [2]int
	err = syscall.Pipe(stdin[:])
	if err == nil {
		err = syscall.Pipe(stdout[:])
	}
	if err != nil {
		return nil, err
	}
	initFlag = append(initFlag, false)

	conn, err := SshConnect(socketName, &SshSvcCfg{initFlag[0], SshSvcConnect, "", ""}, stdin[0], stdout[1])
	if err != nil {
		fmt.Printf("Connect %s failed: %s\n", os.Args[1], err)
		return nil, err
	}
	r := new(ExpecterClient)
	r.SshConn = *conn
	syscall.Close(stdin[0])
	syscall.Close(stdout[1])
	r.src = *fdio.NewTimeoutReader(stdout[0], 30*time.Second)
	r.dest = fdio.FdWriter(stdin[1])

	r.Expecter = *expect.NewExpecter(&r.src)
	return r, nil
}

func (r *ExpecterClient) SetVerbose(verbose bool) {
	r.verbose = verbose
}

func (r *ExpecterClient) SetTimeout(t time.Duration) {
	r.src.SetTimeout(t)
}

func (r *ExpecterClient) Exch(exp, cmd string) error {
	var err error
	if exp != "" {
		if err = r.Expect(exp); err != nil {
			return err
		}
		if r.verbose {
			os.Stdout.Write(r.Payload())
			os.Stdout.Write(r.Match())
		}
	}
	if cmd != "" {
		_, err = r.dest.Write([]byte(cmd+"\r"))
	}
	return err
}

func (r *ExpecterClient) Drain() {
	buf := make([]byte, 512)
	for {
		n, err := r.src.Read(buf)
		if n == 0 || err != nil {
			break
		}
	}
}

func (r *ExpecterClient) Close() {
	r.src.Close()
	r.dest.Close()
	r.SshConn.Close()
}

func (r *ExpecterClient) Interact() {
	r.SttyRaw()
	go func() {
		n, err := io.Copy(r.dest, os.Stdin)
		fmt.Printf("## from tty: %d, %v\r\n", n, err)
	}()
	go func() {
		var n int
		var err error
		for {
			var k int64
			k, err = io.Copy(os.Stdout, fdio.FdReader(r.src.Fd()))
			n += int(k)
			if err.Error() != "timeout" {
				break
			}
		}
		fmt.Printf("## from src: %d, %v\r\n", n, err)
	}()
	r.CloseWait()
}
