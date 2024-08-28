package x509ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/erixzone/crypto/pkg/ed25519"
	"github.com/erixzone/crypto/pkg/x509"
	"github.com/erixzone/crypto/pkg/x509/pkix"
)

// RandInt : return a random integer with the specified number of bits
func RandInt(nbits uint) (*big.Int, error) {
	one := big.NewInt(1)
	return rand.Int(rand.Reader, one.Lsh(one, nbits))
}

var (
	oidX509v3CertificatePolicies = []int{2, 5, 29, 32}
	oidX509v3AnyPolicy           = []int{2, 5, 29, 32, 0}
	oidUserNotice                = []int{1, 3, 6, 1, 5, 5, 7, 2, 2}
)

const (
	tagVisibleString = 26
)

type x509v3Policy struct {
	Oid  asn1.ObjectIdentifier
	Data []asn1.RawValue
}

type x509v3AnyPolicy struct {
	Oid      asn1.ObjectIdentifier
	Policies []x509v3Policy
}

// policyText : store a "User Notice" policy extension made from a string
func policyText(t interface{}, msg string) error {
	if msg == "" {
		return nil
	}
	vs := asn1.RawValue{Tag: tagVisibleString, Bytes: []byte(msg)}
	un := x509v3Policy{Oid: oidUserNotice, Data: []asn1.RawValue{vs}}
	policy := x509v3AnyPolicy{Oid: oidX509v3AnyPolicy, Policies: []x509v3Policy{un}}
	data, err := asn1.Marshal([]x509v3AnyPolicy{policy})
	if err != nil {
		return fmt.Errorf("asn1.Marshal: %s", err)
	}
	e := pkix.Extension{Id: oidX509v3CertificatePolicies, Value: data}
	switch xt := t.(type) {
	case *x509.Certificate:
		xt.ExtraExtensions = append(xt.ExtraExtensions, e)
	case *x509.CertificateRequest:
		xt.ExtraExtensions = append(xt.ExtraExtensions, e)
	default:
		return fmt.Errorf("policyText: unsupported interface type %T", t)
	}
	return nil
}

func makeTemplate(subject pkix.Name, nmonths int, ptext string) (*x509.Certificate, error) {
	var err error

	t := new(x509.Certificate)

	t.SerialNumber, err = RandInt(64)
	if err != nil {
		return nil, fmt.Errorf("RandInt: %s", err)
	}

	t.Subject = subject
	t.NotBefore = time.Now()
	t.NotAfter = t.NotBefore.AddDate(0, nmonths, 0)
	t.BasicConstraintsValid = true
	err = policyText(t, ptext)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func createCertificate(template, parent *x509.Certificate, pub, priv interface{}) (*x509.Certificate, error) {
	pbits, _, err := x509.MarshalPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalPublicKey: %s", err)
	}
	keyid := sha1.Sum(pbits)
	template.SubjectKeyId = keyid[:]
	if parent.AuthorityKeyId == nil {
		template.AuthorityKeyId = template.SubjectKeyId
	}
	DERcert, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("x509.CreateCertificate: %s", err)
	}
	cert, err := x509.ParseCertificate(DERcert)
	if err != nil {
		return nil, fmt.Errorf("x509.ParseCertificate: %s", err)
	}
	return cert, nil
}

// MakeRootCert : make a (self-signed) root certificate with a generated RSA key
func MakeRootCert(subject pkix.Name, nmonths int, ptext string) (*x509.Certificate, *rsa.PrivateKey, error) {
	t, err := makeTemplate(subject, nmonths, ptext)
	if err != nil {
		return nil, nil, fmt.Errorf("makeTemplate: %s", err)
	}
	t.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	t.IsCA = true

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("rsa.GenerateKey: %s", err)
	}
	cert, err := createCertificate(t, t, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("createCertificate: %s", err)
	}
	return cert, priv, nil
}

// MakeCACert : make a CA certificate with a generated ED25519 key
func MakeCACert(subject pkix.Name, nmonths int, ptext string,
	parent *x509.Certificate, parPriv interface{}) (*x509.Certificate, ed25519.PrivateKey, error) {
	t, err := makeTemplate(subject, nmonths, ptext)
	if err != nil {
		return nil, nil, fmt.Errorf("makeTemplate: %s", err)
	}
	t.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	t.IsCA = true

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ed25519.GenerateKey: %s", err)
	}
	cert, err := createCertificate(t, parent, pub, parPriv)
	if err != nil {
		return nil, nil, fmt.Errorf("createCertificate: %s", err)
	}
	return cert, priv, nil
}

// MakeLeafCert : make a leaf certificate with a generated ED25519 key
func MakeLeafCert(subject pkix.Name, nmonths int, ptext string,
	parent *x509.Certificate, parPriv interface{}) (*x509.Certificate, ed25519.PrivateKey, error) {
	t, err := makeTemplate(subject, nmonths, ptext)
	if err != nil {
		return nil, nil, fmt.Errorf("makeTemplate: %s", err)
	}
	t.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ed25519.GenerateKey: %s", err)
	}
	cert, err := createCertificate(t, parent, pub, parPriv)
	if err != nil {
		return nil, nil, fmt.Errorf("createCertificate: %s", err)
	}
	return cert, priv, nil
}

// MakeCSR : make a Certificate Signing Request with a generated ED25519 key
func MakeCSR(subject pkix.Name, ptext string) ([]byte, ed25519.PrivateKey, error) {
	t := new(x509.CertificateRequest)
	t.Subject = subject
	err := policyText(t, ptext)
	if err != nil {
		return nil, nil, err
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("ed25519.GenerateKey: %s", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, t, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("x509.CreateCertificateRequest: %s", err)
	}
	return csr, priv, nil
}

// SignCACert : sign a CSR for an intermediate CA
func SignCACert(csr []byte, nmonths int, parent *x509.Certificate, parPriv interface{}) (*x509.Certificate, error) {
	r, err := x509.ParseCertificateRequest(csr)
	if err != nil {
		return nil, fmt.Errorf("x509.ParseCertificateRequest: %s", err)
	}
	err = r.CheckSignature()
	if err != nil {
		return nil, fmt.Errorf("x509.CheckSignature: %s", err)
	}
	t, err := makeTemplate(r.Subject, nmonths, "")
	if err != nil {
		return nil, fmt.Errorf("makeTemplate: %s", err)
	}

	t.DNSNames = r.DNSNames
	t.EmailAddresses = r.EmailAddresses
	t.IPAddresses = r.IPAddresses
	t.URIs = r.URIs
	t.ExtraExtensions = r.Extensions // I know, they started out as ExtraExtensions

	t.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	t.IsCA = true

	return createCertificate(t, parent, r.PublicKey, parPriv)
}

// CertToPEM : produce a certificate in PEM format
func CertToPEM(cert *x509.Certificate) []byte {
	b := &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	return pem.EncodeToMemory(b)
}

// WriteCertPEMFile : write a certificate to a file in PEM format
func WriteCertPEMFile(cert *x509.Certificate, fname string, perms os.FileMode) error {
	return ioutil.WriteFile(fname, CertToPEM(cert), perms)
}

// KeyToPEM : produce a private key in PEM format
func KeyToPEM(key interface{}) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	b := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return pem.EncodeToMemory(b), nil
}

// WriteKeyPEMFile : write a private key to a file in PEM format
func WriteKeyPEMFile(key interface{}, filename string, perms os.FileMode) error {
	bytes, err := KeyToPEM(key)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, bytes, perms)
}

// CSRToPEM : produce a CSR in PEM format
func CSRToPEM(csr []byte) []byte {
	b := &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}
	return pem.EncodeToMemory(b)
}

// WriteCSRPEMFile : write a CSR to a file in PEM format
func WriteCSRPEMFile(csr []byte, fname string, perms os.FileMode) error {
	return ioutil.WriteFile(fname, CSRToPEM(csr), perms)
}

func readPEMBytes(pemData []byte, tipo string) (*pem.Block, error) {
	asn1, _ := pem.Decode(pemData)
	if asn1 == nil {
		return nil, fmt.Errorf("pem.Decode: failed to parse PEM")
	}
	if tipo != "" && tipo != asn1.Type {
		return nil, fmt.Errorf("readPEMFile: expected \"%s\", got \"%s\"",
			tipo, asn1.Type)
	}
	return asn1, nil
}

// ReadCertPEMBytes : read a certificate from a byte slice in PEM format
func ReadCertPEMBytes(pemData []byte) (*x509.Certificate, error) {
	asn1, err := readPEMBytes(pemData, "CERTIFICATE")
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(asn1.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ParseCertificate: %s", err)
	}
	return cert, nil
}

// ReadCertPEMFile : read a certificate from a file in PEM format
func ReadCertPEMFile(filename string) (*x509.Certificate, error) {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ReadFile %s: %s", filename, err)
	}
	return ReadCertPEMBytes(pemData)
}

// ReadKeyPEMBytes : read a private key from a byte slice in PEM format
func ReadKeyPEMBytes(pemData []byte) (interface{}, error) {
	asn1, err := readPEMBytes(pemData, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(asn1.Bytes)
	if err != nil {
		return nil, fmt.Errorf("ParsePKCS8PrivateKey: %s", err)
	}
	return key, nil
}

// ReadKeyPEMFile : read a private key from a file in PEM format
func ReadKeyPEMFile(filename string) (interface{}, error) {
	pemData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ReadFile %s: %s", filename, err)
	}
	return ReadKeyPEMBytes(pemData)
}

// MakeCertPools : assemble root and intermediate cert's
func MakeCertPools(certs []*x509.Certificate) *x509.VerifyOptions {
	rootPool := x509.NewCertPool()
	intPool := x509.NewCertPool()
	for _, cert := range certs {
		if cert.Issuer.String() == cert.Subject.String() {
			rootPool.AddCert(cert)
		} else {
			intPool.AddCert(cert)
		}
	}
	return &x509.VerifyOptions{Roots: rootPool, Intermediates: intPool}
}

// ReadCertPools : assemble root and intermediate cert's
func ReadCertPools(filenames []string) (*x509.VerifyOptions, error) {
	var certs []*x509.Certificate
	for _, fname := range filenames {
		cert, err := ReadCertPEMFile(fname)
		if err != nil {
			return nil, fmt.Errorf("ReadCertPools: %s: %s", fname, err)
		}
		certs = append(certs, cert)
	}
	return MakeCertPools(certs), nil
}
