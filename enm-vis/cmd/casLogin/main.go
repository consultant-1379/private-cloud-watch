package main

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/casbah"
)

func _main() int {
	cas, err := casbah.NewCasbah(false)
	if err != nil {
		fmt.Printf("Casbah failed: %s\n", err)
		return 1
	}
	defer cas.Close()

	cas.Interact()
	return 0
}

func main() {
	os.Exit(_main())
}
