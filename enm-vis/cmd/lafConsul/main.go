package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/casbah"
)

func _main() int {
	cas, err := casbah.NewCasbah(false)
	if err != nil {
		fmt.Printf("Casbah failed: %s\n", err)
		return 1
	}
	defer cas.Close()
//	cas.Debug(os.Stdout)
	cas.SetTimeout(60*time.Second)

	cas.Exch("", fmt.Sprintf("ssh %s cloud-user", os.Args[1]))
	cas.Exch("cloud-user@[0-9.]+'s password: ", cas.Pwd("cloud-user"))

	prompt := "(\033\\][^\007]*\007)?"
	prompt += `\[cloud-user@[^ ]+ ~\]\$ `
	cas.Exch(prompt, "stty -echo")
	cas.Exch(prompt, "curl http://127.0.0.1:8500/v1/agent/members")
	cas.Exch(prompt, "")

	jbytes := cas.CopyPayload()

	cas.Exch("", "exit") // to cas
	cas.Exch(" > ", "exit") // to ecgw
	cas.Exch(" > ", "exit") // home

	fmt.Printf("data len %d\n", len(jbytes))
	err = ioutil.WriteFile("json.out", jbytes, 0644)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	return 0
}

func main() {
	os.Exit(_main())
}
