package main

import (
	"fmt"
	"os"
	"time"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
)

func Main(sockname string) error {
	var ecgw, casgw string
	var err error
	var ok bool
	if ecgw, ok = os.LookupEnv("ECGW"); !ok {
		ecgw = "eusecgw"
	}
	if casgw, ok = os.LookupEnv("CASGW"); !ok {
		casgw = "at1-nmaas1-cas1"
	}

	exp, err := client.NewExpecterClient(sockname, true)
	if err != nil {
		return err
	}
	//exp.SetVerbose(true)

	prompt := fmt.Sprintf(`%s(\.[a-z0-9]+)* > `, ecgw)
	err = exp.Exch(prompt, "date")
	if err != nil {
		return err
	}
	err = exp.Exch(prompt, fmt.Sprintf("ssh %s", casgw))
	if err != nil {
		return err
	}

	prompt = fmt.Sprintf("%s > ", casgw)
	err = exp.Exch(prompt, "date")
	if err != nil {
		return err
	}
	err = exp.Exch(prompt, "")
	if err != nil {
		return err
	}

	go func() {
		for {
			time.Sleep(time.Duration(120) * time.Second)
			exp.Exch("", "date")
		}
	}()
	go exp.Drain()

	return nil
}

func main() {
	fmt.Printf("This is not the program you're looking for.\n")
}
