package main

import (
	"github.com/erixzone/crypto/pkg/x509/pkix"
	"github.com/erixzone/myriad/pkg/x509ca"
)

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	subject := pkix.Name{
		Organization: []string{"Erixzone"},
		CommonName:   "Erixzone Root Certificate",
	}
	root, rootPriv, err := x509ca.MakeRootCert(subject, 120,
		"If you trust this Root Certificate then we have a bridge that should interest you.")
	panicIf(err)
	err = x509ca.WriteCertPEMFile(root, "Root.crt", 0644)
	panicIf(err)
	err = x509ca.WriteKeyPEMFile(rootPriv, "Root.key", 0600)
	panicIf(err)

	subject.CommonName = "Erixzone Server CA X1"
	CA, caPriv, err := x509ca.MakeCACert(subject, 36,
		"This Certificate Authority is for entertainment purposes only.",
		root, rootPriv)
	panicIf(err)
	err = x509ca.WriteCertPEMFile(CA, "CA.crt", 0644)
	panicIf(err)
	err = x509ca.WriteKeyPEMFile(caPriv, "CA.key", 0600)
	panicIf(err)
}
