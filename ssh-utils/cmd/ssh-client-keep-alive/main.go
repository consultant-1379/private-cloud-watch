package main

import (
	"fmt"
	"os"
	"time"
	"strconv"
	"syscall"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
)

func _main() int {
	var err error
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s addr [secs]\n", os.Args[0])
		return 127
	}
	secs := 60
	if len(os.Args) > 2 {
		secs, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("%s\n", err)
			return 126
		}
	}
	var stdin [2]int
	err = syscall.Pipe(stdin[:])
	if err != nil {
		fmt.Printf("Pipe failed: %s\n", err)
		return 2
	}
	conn, err := client.Connect(os.Args[1], stdin[0])
	if err != nil {
		fmt.Printf("Connect %s failed: %s\n", os.Args[1], err)
		return 1
	}
	defer conn.Close()

	go func() {
		for {
			time.Sleep(time.Duration(secs) * time.Second)
			syscall.Write(stdin[1], []byte("date\r"))
		}
	}()

	conn.CloseWait()
	return 0
}

func main() {
	os.Exit(_main())
}
