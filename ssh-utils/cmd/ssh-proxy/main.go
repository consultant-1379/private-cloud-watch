package main

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
)

func _main() int {
	if len(os.Args) != 3 {
		fmt.Printf("usage: %s addr dialstr\n", os.Args[0])
		return 127
	}
	conn, err := client.Forward(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Printf("Forward %s %s failed: %s\n", os.Args[1], os.Args[2], err)
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
