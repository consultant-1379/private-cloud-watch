package main

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/casbah"
)

func _main() int {
	cas, err := casbah.NewCasbah(true)
	if err != nil {
		fmt.Printf("Casbah failed: %s\n", err)
		return 1
	}
	defer cas.Close()

	cas.Exch("", "ssh 10.2.10.10 cloud-user")
	cas.Exch("cloud-user@10.2.10.10's password: ", cas.Pwd("cloud-user"))

	prompt := `\[cloud-user@staging01-vnflaf-services-0 ~\]\$ `
	cas.Exch(prompt, "date")
	cas.Exch(prompt, "")

	cas.Interact()
	return 0
}

func main() {
	os.Exit(_main())
}
