package pwd

import (
	"errors"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

type PwdCache map[string]string

func GetPwds(pwdfile string) (PwdCache, error) {
	if pwdfile == "" {
		var ok bool
		pwdfile, ok = os.LookupEnv("PWDFILE")
		if !ok {
			return nil, errors.New("missing PWDFILE parameter")
		}
	}
	data, err := ioutil.ReadFile(pwdfile)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	pwds := make(map[string]string)
	for {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			break
		}
		kv := bytes.SplitN(line[:len(line)-1], []byte("\t"), 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("cannot parse %s", pwdfile)
		}
		pwds[string(kv[0])] = string(kv[1])
	}
	return PwdCache(pwds), nil
}

func (p PwdCache) Pwd(k string) string {
	return p[k]
}
