package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/casbah"
)

func usage() {
	fmt.Printf("usage: %s sockname user@host[:port] cmd...\n", os.Args[0])
	os.Exit(127)
}

func _main() int {
	if len(os.Args) < 4 {
		usage()
	}
	args := strings.SplitN(os.Args[2], "@", 2)
	if len(args) != 2 {
		usage()
	}

	session, err :=  casbah.NewProxySession(os.Args[1], args[0], args[1])
	if err != nil {
		fmt.Printf("NewProxySession failed: %s\n", err)
		return 1
	}
	defer session.Close()
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Run(strings.Join(os.Args[3:], " "))
	return 0
}

func main() {
	os.Exit(_main())
}
