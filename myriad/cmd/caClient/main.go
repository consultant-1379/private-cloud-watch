package main

import (
	"log"

	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/crypto/pkg/x509/pkix"
	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/x509ca"
)

var url = "http://localhost:8666/jsonrpc/"

func main() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("client: ")
	userId := "Vae7fag6ahBo3ahJoh7"
	passwd := "oongei2CheiKaoyiedu"
	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux Testbed"},
		CommonName:         "Crux Testbed CA",
	}
	csr, blocPriv, err := x509ca.MakeCSR(subject, "Without trust there can be no betrayal.")
	if err != nil {
		log.Fatalf("MakeCSR: %s", err.Error())
	}
	ca, err := myriadca.RequestCert(url, userId, passwd, csr)
	if err != nil {
		log.Fatalf("RequestCert: %s", err.Error())
	}
	ca.Last().PrivKey = blocPriv
	rootPool := x509.NewCertPool()
	intPool := x509.NewCertPool()
	rootPool.AddCert(ca.Chain[0].Cert)
	for _, c := range ca.Chain[1 : len(ca.Chain)-1] {
		intPool.AddCert(c.Cert)
	}
	opts := x509.VerifyOptions{Roots: rootPool, Intermediates: intPool}
	vcs, err := ca.Last().Cert.Verify(opts)
	if err != nil {
		log.Fatalf("Verify: %s", err)
	}
	for i, ch := range vcs {
		log.Printf("%d: %d\n", i, len(ch))
		for j, c := range ch {
			log.Printf("%2d: %s %s", j, c.SerialNumber.Text(16), c.Subject)
		}
	}
}
