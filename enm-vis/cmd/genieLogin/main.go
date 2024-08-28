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

	cas.Exch("", "ssh 192.168.255.248 bucinwci")
	cas.Exch("bucinwci@192.168.255.248's password: ", cas.Pwd("bucinwci"))

	prompt := `\[bucinwci@genie-utility ~\]\$ `
	cas.Exch(prompt, "date")
	cas.Exch(prompt, "")

	cas.Interact()
	return 0
}

func main() {
	os.Exit(_main())
}
