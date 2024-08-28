package casbah

import (
	"fmt"
	"os"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/client"
	"github.com/erixzone/xaas/platform/ssh-utils/pkg/pwd"
)

type Casbah struct {
	client.ExpecterClient
	pwd.PwdCache
}

func NewCasbah(verbose bool) (*Casbah, error) {
	var ecgw, casgw string
	var err error
	var ok bool
	cas := new(Casbah)
	cas.PwdCache, err = pwd.GetPwds("")
	if err != nil {
		return nil, err
	}
	if ecgw, ok = os.LookupEnv("ECGW"); !ok {
		ecgw = "eusecgw"
	}
	if casgw, ok = os.LookupEnv("CASGW"); !ok {
		casgw = "at1-nmaas1-cas1"
	}
	exp, err := client.NewExpecterClient(ecgw)
	if err != nil {
		return nil, err
	}
	cas.ExpecterClient = *exp
	cas.SetVerbose(verbose)

	prompt := fmt.Sprintf(`%s(\.[a-z0-9]+)* > `, ecgw)
	cas.Exch(prompt, "date")
	cas.Exch(prompt, fmt.Sprintf("ssh %s", casgw))

	prompt = fmt.Sprintf("%s > ", casgw)
	cas.Exch(prompt, "date")
	cas.Exch(prompt, "")

	return cas, nil
}
