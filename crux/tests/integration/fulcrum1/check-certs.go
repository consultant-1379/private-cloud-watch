package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/erixzone/crux/pkg/x509ca"
	"github.com/erixzone/crypto/pkg/ed25519"
)

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func main () {
	certDir := "./"
	fp, err := os.Open(certDir)
	panicOn(err)
	dirlist, err := fp.Readdirnames(0)
	panicOn(err)
	err = fp.Close()
	panicOn(err)
	keyFile := ""
	for _, f := range dirlist {
		if !strings.HasSuffix(f, ".key") {
			continue
		}
		if keyFile != "" {
			panicOn(fmt.Errorf("multiple .key files in %s", certDir))
		}
		keyFile = f
	}
	if keyFile == "" {
		panicOn(fmt.Errorf("no .key file in %s", certDir))
	}
	var caCerts []string
	for _, f := range dirlist {
		if strings.HasSuffix(f, ".crt") &&
			!strings.HasPrefix(f, keyFile[:len(keyFile)-3]) {
			caCerts = append(caCerts, certDir+f)
		}
	}
	if len(caCerts) == 0 {
		panicOn(fmt.Errorf("no CA certs in %s", certDir))
	}
	keyFile = certDir + keyFile
	opts, err := x509ca.ReadCertPools(caCerts)
	panicOn(err)
	leaf, err := x509ca.ReadCertPEMFile(keyFile[:len(keyFile)-3] + "crt")
	panicOn(err)

	chains, err := leaf.Verify(*opts)
	panicOn(err)
	for i, chain := range chains {
		fmt.Printf("chain %d: length %d\n", i, len(chain))
		for j, cert := range chain {
			fmt.Printf("%2d: %s %s\n", j, cert.SerialNumber.Text(16), cert.Subject)
		}
	}
	privkey, err := x509ca.ReadKeyPEMFile(keyFile)
	panicOn(err)
	pubkey := privkey.(ed25519.PrivateKey).Public()
	if !bytes.Equal(pubkey.(ed25519.PublicKey), leaf.PublicKey.(ed25519.PublicKey)) {
		panic(fmt.Errorf("keys do not match"))
	}
}
