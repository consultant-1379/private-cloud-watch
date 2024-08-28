package x509ca

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/erixzone/crypto/pkg/ed25519"
	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/crypto/pkg/x509/pkix"
)

func writeCert(cert *x509.Certificate, priv interface{}, filestem string) error {
	err := WriteCertPEMFile(cert, filestem+".crt", 0644)
	if err != nil {
		return fmt.Errorf("WriteCertPEMFile: %s", err)
	}
	err = WriteKeyPEMFile(priv, filestem+".key", 0600)
	if err != nil {
		return fmt.Errorf("WriteKeyPEMFile: %s", err)
	}
	return nil
}

func readCert(filestem string) (*x509.Certificate, interface{}, error) {
	cert, err := ReadCertPEMFile(filestem + ".crt")
	if err != nil {
		return nil, nil, fmt.Errorf("ReadCertPEMFile: %s", err)
	}
	priv, err := ReadKeyPEMFile(filestem + ".key")
	if err != nil {
		return nil, nil, fmt.Errorf("ReadKeyPEMFile: %s", err)
	}
	return cert, priv, nil
}

func writeCA() error {
	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux"},
		Locality:           []string{"Brookside"},
		Province:           []string{"NJ"},
		CommonName:         "Erixzone crux Root Certificate",
	}
	root, rootPriv, err := MakeRootCert(subject, 120,
		"If you trust this Root Certificate then we have a bridge that should interest you.")
	if err != nil {
		return fmt.Errorf("MakeRootCert: %s", err)
	}
	err = writeCert(root, rootPriv, "goRoot")
	if err != nil {
		return fmt.Errorf("writeCert (root): %s", err)
	}

	subject.CommonName = "Erixzone crux Server CA X1"
	CA, caPriv, err := MakeCACert(subject, 36,
		"This Certificate Authority is for entertainment purposes only.",
		root, rootPriv)
	if err != nil {
		return fmt.Errorf("MakeCACert: %s", err)
	}
	err = writeCert(CA, caPriv, "goCA")
	if err != nil {
		return fmt.Errorf("writeCert (CA): %s", err)
	}
	return nil
}

func writeLeaf(leafname string, CA *x509.Certificate, caPriv interface{}, retPriv *ed25519.PrivateKey) error {
	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux"},
		DomainComponent:    []string{"Ripstop", "Myriad"},
		CommonName:         leafname,
	}
	leaf, leafPriv, err := MakeLeafCert(subject, 3,
		"Keep away from children.  This certificate is a toy.",
		CA, caPriv)
	if err != nil {
		return fmt.Errorf("MakeLeafCert (%s): %s", leafname, err)
	}
	err = writeCert(leaf, leafPriv, leafname)
	if err != nil {
		return fmt.Errorf("writeCert (%s): %s", leafname, err)
	}
	if retPriv != nil {
		*retPriv = leafPriv
	}
	return nil
}

func testCSR(leafname string, CA *x509.Certificate, caPriv interface{}, opts *x509.VerifyOptions) error {
	subject := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"Erixzone"},
		OrganizationalUnit: []string{"Crux Testbed"},
		CommonName:         "Crux Testbed CA",
	}
	csr, blocPriv, err := MakeCSR(subject, "Without trust there can be no betrayal.")
	if err != nil {
		return err
	}
	blocCA, err := SignCACert(csr, 12, CA, caPriv)
	if err != nil {
		return err
	}
	err = writeCert(blocCA, blocPriv, "blocCA")
	if err != nil {
		return fmt.Errorf("writeCert (root): %s", err)
	}
	err = writeLeaf(leafname, blocCA, blocPriv, nil)
	if err != nil {
		return err
	}
	opts.Intermediates.AddCert(blocCA)
	return nil
}

func verifyCert(node string, opts *x509.VerifyOptions, cuckoo bool) error {
	hcert, err := ReadCertPEMFile(node + ".crt")
	if err != nil {
		return fmt.Errorf("verifyCert: read %s: %s", node, err)
	}
	chains, err := hcert.Verify(*opts)
	if err != nil {
		if cuckoo {
			fmt.Printf("verifyCert (cuckoo): verify %s: %s\n", node, err)
		} else {
			return fmt.Errorf("verifyCert: verify %s: %s", node, err)
		}
	}
	for i, chain := range chains {
		fmt.Printf("%d: %d\n", i, len(chain))
		for j, cert := range chain {
			fmt.Printf("%2d: %s %s\n", j, cert.SerialNumber.Text(16), cert.Subject)
		}
	}
	fmt.Printf("\n")
	if err == nil && cuckoo {
		return fmt.Errorf("verifyCert: verify %s: cuckoo cert passed", node)
	}
	return nil
}

func TestMakeCerts(t *testing.T) {
	err := writeCA()
	if err != nil {
		t.Errorf("writeCA: %s", err)
	}
	CA, caPriv, err := readCert("goCA")
	if err != nil {
		t.Errorf("readCert (CA): %s", err)
	}

	nodes := []string{"flockA", "flockB", "flockC"}

	leafPriv := ed25519.PrivateKey{}
	privP := &leafPriv
	_, signPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Errorf("ed25519.GenerateKey (cuckoo): %s", err)
	}
	for _, name := range nodes {
		if writeLeaf(name, CA, signPriv, privP) != nil {
			t.Errorf("writeLeaf (%s): %s", name, err)
		}
		privP = nil
		signPriv = caPriv.(ed25519.PrivateKey)
	}

	data, err := ReadKeyPEMFile(nodes[0] + ".key")
	if err != nil {
		t.Errorf("ReadKeyPEMFile (%s): %s", nodes[0], err)
	}
	readPriv := data.(ed25519.PrivateKey)

	if !bytes.Equal(leafPriv[:], readPriv[:]) {
		t.Error("ed25519 private key recovery failed")
	}

	opts, err := ReadCertPools([]string{"goRoot.crt", "goCA.crt"})
	if err != nil {
		t.Errorf("ReadCertPools (CA): %s", err)
	}
	err = testCSR("flockD", CA, signPriv, opts)
	if err != nil {
		t.Errorf("testCSR: %s", err)
	}
	nodes = append(nodes, "flockD")
	for k, node := range nodes {
		if verifyCert(node, opts, k == 0) != nil {
			t.Errorf("verifyCert (%s): %s", node, err)
		}
	}
}
