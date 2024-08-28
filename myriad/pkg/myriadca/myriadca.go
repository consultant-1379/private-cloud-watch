package myriadca

import (
	"archive/tar"
	"bytes"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/crypto/pkg/x509/pkix"
	"github.com/erixzone/myriad/pkg/x509ca"
	"github.com/spf13/viper"
)

// Certriplicate : the three things that you always need
type Certriplicate struct {
	Cert    *x509.Certificate
	PrivKey interface{} // nil if we don't have it
	certPEM []byte      // nil until we need it
}

// PEM : lazy accessor
func (ct *Certriplicate) PEM() []byte {
	if ct.certPEM == nil {
		ct.certPEM = x509ca.CertToPEM(ct.Cert)
	}
	return ct.certPEM
}

// CertificateAuthority : chain of Certriplicate's, with methods
type CertificateAuthority struct {
	HasLeaf bool             // true if Chain ends with a leaf certificate
	Chain   []*Certriplicate // [0] is the root
}

// Append : add to the chain
func (ca *CertificateAuthority) Append(cert *x509.Certificate, privKey interface{}) {
	ca.Chain = append(ca.Chain, &Certriplicate{cert, privKey, nil})
}

// Last : leaf certificate or signing certificate
func (ca *CertificateAuthority) Last() *Certriplicate {
	if len(ca.Chain) == 0 {
		return nil
	}
	return ca.Chain[len(ca.Chain)-1]
}

// MakeCertificateAuthorityTopLevels : out of the vacuum
func MakeCertificateAuthorityTopLevels() (*CertificateAuthority, error) {
	ca := new(CertificateAuthority)
	subject := pkix.Name{
		Organization: []string{"Erixzone"},
		CommonName:   "Erixzone Root Certificate",
	}
	rootCert, rootPriv, err := x509ca.MakeRootCert(subject, 120,
		"If you trust this Root Certificate then we have a bridge that will interest you.")
	if err != nil {
		return nil, fmt.Errorf("MakeRootCert: %s", err)
	}
	ca.Append(rootCert, rootPriv)

	subject.CommonName = "Erixzone crux Server CA X1"
	caCert, caPriv, err := x509ca.MakeCACert(subject, 36,
		"This Certificate Authority is for entertainment purposes only.",
		rootCert, rootPriv)
	if err != nil {
		return nil, fmt.Errorf("MakeCACert: %s", err)
	}
	ca.Append(caCert, caPriv)

	return ca, nil
}

// MakeCertificateAuthority : out of the vacuum
func MakeCertificateAuthority() (*CertificateAuthority, error) {
	ca, err := MakeCertificateAuthorityTopLevels()
	if err != nil {
		return nil, err
	}
	subject := ca.Last().Cert.Subject
	subject.CommonName = "Crux Bloc CA"
	subject.OrganizationalUnit = []string{viper.GetString("cert_OU")}
	subject.DomainComponent = strings.Split(viper.GetString("docker.network.name"), ".")
	blocCert, blocPriv, err := x509ca.MakeCACert(subject, 36,
		"Without trust there can be no betrayal.",
		ca.Last().Cert, ca.Last().PrivKey)
	if err != nil {
		return nil, fmt.Errorf("MakeCACert (bloc): %s", err)
	}
	ca.Append(blocCert, blocPriv)
	return ca, nil
}

// FetchCertificateAuthority : from an external server
func FetchCertificateAuthority() (*CertificateAuthority, error) {
	subject := pkix.Name{}
	subject.Organization = []string{"Erixzone"}
	subject.OrganizationalUnit = []string{viper.GetString("cert_OU")}
	subject.CommonName = "Crux Bloc CA"
	subject.DomainComponent = strings.Split(viper.GetString("docker.network.name"), ".")
	csr, blocPriv, err := x509ca.MakeCSR(subject, "Without trust there can be no betrayal.")
	if err != nil {
		return nil, fmt.Errorf("MakeCSR: %s", err.Error())
	}
	ca, err := RequestCert(viper.GetString("ca.url"),
		viper.GetString("ca.userid"), viper.GetString("ca.passwd"), csr)
	if err != nil {
		return nil, fmt.Errorf("RequestCert: %s", err.Error())
	}
	ca.Last().PrivKey = blocPriv
	return ca, nil
}

// MakeLeafCert : rhubarb
func (ca *CertificateAuthority) MakeLeafCert(leafname string) (*CertificateAuthority, error) {
	var err error
	if ca.HasLeaf {
		return nil, fmt.Errorf("MakeLeafCert: CA has leaf")
	}
	signer := ca.Last()
	subject := signer.Cert.Subject
	subject.CommonName = leafname

	leafCert, leafKey, err := x509ca.MakeLeafCert(subject, 3,
		"Keep away from children.  This certificate is a toy.",
		signer.Cert, signer.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("MakeLeafCert (%s): %s", leafname, err)
	}
	leaf := new(CertificateAuthority)
	leaf.Chain = append(leaf.Chain, ca.Chain...)
	leaf.Append(leafCert, leafKey)
	leaf.HasLeaf = true
	return leaf, nil
}

// TarStream : export certificate chain plus local private key
func (leaf *CertificateAuthority) TarStream() (*bytes.Buffer, error) {
	if !leaf.HasLeaf {
		return nil, fmt.Errorf("TarStream: CA has no leaf")
	}
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	dir := path.Clean(viper.GetString("certdir"))
	if len(dir) > 0 && dir[0] == '/' { // Path is "/" in UploadToContainerOptions
		dir = dir[1:]
	}
	if len(dir) > 0 {
		elems := strings.Split(dir, "/")
		dir = ""
		for _, e := range elems {
			dir += e + "/"
			err := addToTar(tw, dir, 0755, nil)
			if err != nil {
				return nil, err
			}
		}
	}
	filestem := dir + leaf.Last().Cert.Subject.CommonName
	for i, ct := range leaf.Chain {
		var fname string
		switch i {
		case 0:
			fname = dir + "Root.crt"
		case len(leaf.Chain) - 2:
			fname = dir + "CA.crt"
		case len(leaf.Chain) - 1:
			fname = filestem + ".crt"
		default:
			fname = fmt.Sprintf("%sInter%d.crt", dir, i)
		}
		err := addToTar(tw, fname, 0444, ct.PEM())
		if err != nil {
			return nil, err
		}
	}
	data, err := x509ca.KeyToPEM(leaf.Last().PrivKey)
	if err != nil {
		return nil, err
	}
	err = addToTar(tw, filestem+".key", 0400, data)
	if err != nil {
		return nil, err
	}
	tw.Close()
	return buf, nil
}

func addToTar(tw *tar.Writer, name string, mode int, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    int64(mode),
		Size:    int64(len(data)),
		Uid:     viper.GetInt("docker.uid"),
		Gid:     viper.GetInt("docker.gid"),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if data != nil {
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// LeafTarStream : rhubarb
func (ca *CertificateAuthority) LeafTarStream(leafname string) (*bytes.Buffer, error) {
	cert, err := ca.MakeLeafCert(leafname)
	if err != nil {
		return nil, err
	}
	return cert.TarStream()
}
