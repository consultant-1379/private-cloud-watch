package main

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
)

func _main() int {
	exp, err := client.NewExpecterClient(os.Args[1])
	if err != nil {
		fmt.Printf("NewExpecterClient %s failed: %s\n", os.Args[1], err)
		return 1
	}
	defer exp.Close()
	exp.SetVerbose(true)
	prompt := `\[[a-z]+@[a-z]+ ~\]\$ `

	exp.Exch(prompt, "date")
	exp.Exch(prompt, "pwd")
	exp.Exch(prompt, "")
	exp.Interact()
	return 0
}

func main() {
	os.Exit(_main())
}
