package ec25519

import (
	"fmt"
	"testing"

	"github.com/erixzone/crux/pkg/x509ca"
	"github.com/erixzone/crypto/pkg/ed25519"
	"github.com/erixzone/crypto/pkg/x509/pkix"
)

func TestEc25519(t *testing.T) {
	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux"},
		Locality:           []string{"Brookside"},
		Province:           []string{"NJ"},
		CommonName:         "Erixzone crux Root Certificate",
	}
	root, rootPriv, err := x509ca.MakeRootCert(subject, 120,
		"If you trust this Root Certificate then we have a bridge that should interest you.")
	if err != nil {
		t.Errorf("MakeRootCert: %s", err)
	}
	subject.CommonName = "Erixzone crux Server CA X1"
	CA, caPriv, err := x509ca.MakeCACert(subject, 36,
		"This Certificate Authority is for entertainment purposes only.",
		root, rootPriv)
	if err != nil {
		t.Errorf("MakeCACert: %s", err)
	}
	leafname := "flock-node-A"
	subject = pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux"},
		DomainComponent:    []string{"Ripstop", "Myriad"},
		CommonName:         leafname,
	}
	leaf, leafPriv, err := x509ca.MakeLeafCert(subject, 3,
		"Keep away from children.  This certificate is a toy.",
		CA, caPriv)
	if err != nil {
		t.Errorf("MakeLeafCert (%s): %s", leafname, err)
	}
	fmt.Printf("leaf public  key = %02x\n", leaf.PublicKey)
	fmt.Printf("leaf private key = %02x\n", leafPriv[:32])
	var ecPub Key
	if err = ed25519.ECPublic(ecPub[:], leaf.PublicKey.(ed25519.PublicKey)); err != nil {
		t.Errorf("ed25519.ECPublic: %s", err)
	}
	fmt.Printf("leaf ec pub  key = %02x\n", ecPub)

	fmt.Printf("key length = %d\n", Keylen)

	var secrA, pubA Key
	err = NewKeyPairFromSeed(leafPriv, &secrA, &pubA)
	if err != nil {
		t.Fail()
	}
	fmt.Printf("secret key A: %x\n", secrA)
	fmt.Printf("public key A: %x\n", pubA)

	if pubA != ecPub {
		t.Fail()
	}

	var secrB, pubB Key
	err = NewKeyPair(&secrB, &pubB)
	if err != nil {
		t.Fail()
	}
	fmt.Printf("secret key B: %x\n", secrB)
	fmt.Printf("public key B: %x\n", pubB)

	var shareAB, shareBA Key
	ScalarMult(&shareAB, &secrA, &pubB)
	ScalarMult(&shareBA, &secrB, &pubA)
	//shareAB[0] ^= 0xff
	fmt.Printf("shared    AB: %x\n", shareAB)
	fmt.Printf("shared    BA: %x\n", shareBA)

	if shareAB != shareBA {
		t.Fail()
	}
}
