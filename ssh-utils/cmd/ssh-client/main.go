package main

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
)

func _main() int {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s addr\n", os.Args[0])
		return 127
	}
	conn, err := client.Connect(os.Args[1])
	if err != nil {
		fmt.Printf("Connect %s failed: %s\n", os.Args[1], err)
		return 1
	}
	defer conn.Close()
	conn.SttyRaw()
	conn.CloseWait()
	return 0
}

func main() {
	os.Exit(_main())
}
