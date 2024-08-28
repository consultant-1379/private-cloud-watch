package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/erixzone/crux/pkg/x509ca"
	"github.com/erixzone/crypto/pkg/ed25519"
)

func main () {
	tarFiles := []string{"./admincert.tar"}
	if len(os.Args) > 1 {
		tarFiles = os.Args[1:]
	}
	failCount := 0
	for _, tarFile := range tarFiles {
		opts, privKey, leaf, err := x509ca.ReadCertsTar(tarFile)
		if err != nil {
			failCount++
			fmt.Printf("read %s: %s\n", tarFile, err)
			continue
		}
		chains, err := leaf.Verify(*opts)
		if err != nil {
			failCount++
			fmt.Printf("verify %s: %s\n", tarFile, err)
			continue
		}
		for i, chain := range chains {
			fmt.Printf("%s: chain %d: length %d\n", tarFile, i, len(chain))
			for j, cert := range chain {
				fmt.Printf("%2d: %s %s\n", j, cert.SerialNumber.Text(16), cert.Subject)
			}
		}
		pubkey := privKey.(ed25519.PrivateKey).Public()
		if !bytes.Equal(pubkey.(ed25519.PublicKey), leaf.PublicKey.(ed25519.PublicKey)) {
			failCount++
			fmt.Printf("%s: leaf public keys do not match\n", tarFile)
		}
	}
	os.Exit(failCount)
}
