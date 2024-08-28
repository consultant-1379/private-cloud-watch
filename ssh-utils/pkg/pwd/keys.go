package pwd

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func GetKeys(keyfiles ...string) ([]ssh.Signer, error) {
	if keyfiles == nil {
		parm, ok := os.LookupEnv("KEYFILES")
		if !ok {
			return nil, errors.New("missing KEYFILES parameter")
		}
		keyfiles = strings.Split(parm, ":")
		if len(keyfiles) == 0 {
			return nil, errors.New("empty KEYFILES parameter")
		}
	}
	var keyring []ssh.Signer
	for _, fname := range keyfiles {
		key, err := ioutil.ReadFile(fname)
		if err != nil {
			return nil, fmt.Errorf("can't read private key: %s", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("can't parse private key: %s", err)
		}
		keyring = append(keyring, signer)
	}
	return keyring, nil
}

func GetKeyDict(username, dialstr, dictfile string) ([]ssh.Signer, error) {
	if dictfile == "" {
		var ok bool
		dictfile, ok = os.LookupEnv("KEYDICT")
		if !ok {
			return nil, errors.New("missing KEYDICT parameter")
		}
	}
	lines, err := ioutil.ReadFile(dictfile)
	if err != nil {
		return nil, fmt.Errorf("can't read dictfile: %s", err)
	}
	var keyring []ssh.Signer
	for lineno, line := range bytes.Split(lines, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		parms := strings.Fields(string(line))
		if len(parms) != 3 {
			return nil, fmt.Errorf("can't parse %s line %d",
				dictfile, lineno+1)
		}
		if username == parms[0] && dialstr == parms[1] {
			keys, err := GetKeys(parms[2])
			if err != nil {
				return nil, err
			}
			keyring = append(keyring, keys...)
		}
	}
	return keyring, nil
}
