package x509ca

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/erixzone/crypto/pkg/tls"
	"github.com/erixzone/crypto/pkg/x509"
)

type bailErr struct {
	err error
}

func bailOn(err error) {
	if err != nil {
		panic(bailErr{err})
	}
}

func bailOut(x interface{}) error {
	if x == nil {
		return nil
	}
	if berr, ok := x.(bailErr); ok {
		return berr.err
	}
	panic(x)
}

// ReadCerts : fetch certificates from the given directory
func ReadCerts(certDir string) (opts *x509.VerifyOptions, privKey interface{}, leaf *x509.Certificate, err error) {
	defer func() {
		err = bailOut(recover())
	}()
	if certDir == "" {
		certDir = "/crux/crt/"
	}
	if !strings.HasSuffix(certDir, "/") {
		certDir += "/"
	}
	fp, err := os.Open(certDir)
	bailOn(err)
	dirlist, err := fp.Readdirnames(0)
	bailOn(err)
	err = fp.Close()
	bailOn(err)
	keyFile := ""
	for _, f := range dirlist {
		if !strings.HasSuffix(f, ".key") {
			continue
		}
		if keyFile != "" {
			bailOn(fmt.Errorf("multiple .key files in"))
		}
		keyFile = f
	}
	if keyFile == "" {
		bailOn(fmt.Errorf("no .key file"))
	}
	var caCerts []string
	for _, f := range dirlist {
		if strings.HasSuffix(f, ".crt") &&
			!strings.HasPrefix(f, keyFile[:len(keyFile)-3]) {
			caCerts = append(caCerts, certDir+f)
		}
	}
	if len(caCerts) == 0 {
		bailOn(fmt.Errorf("no CA certs"))
	}
	keyFile = certDir + keyFile
	opts, err = ReadCertPools(caCerts)
	bailOn(err)
	leaf, err = ReadCertPEMFile(keyFile[:len(keyFile)-3] + "crt")
	bailOn(err)
	privKey, err = ReadKeyPEMFile(keyFile)
	bailOn(err)

	return opts, privKey, leaf, nil
}

// ReadCertsTar : fetch certificates from the given tar archive file
func ReadCertsTar(tarFile string) (opts *x509.VerifyOptions, privKey interface{}, leaf *x509.Certificate, err error) {
	fp, err := os.Open(tarFile)
	if err == nil {
		defer fp.Close()
		return readCertsTar(fp)
	}
	return
}

// ReadCertsTarBytes : fetch certificates from the given tar archive byte slice
func ReadCertsTarBytes(data []byte) (opts *x509.VerifyOptions, privKey interface{}, leaf *x509.Certificate, err error) {
	return readCertsTar(bytes.NewReader(data))
}

func readCertsTar(fp io.Reader) (opts *x509.VerifyOptions, privKey interface{}, leaf *x509.Certificate, err error) {
	defer func() {
		err = bailOut(recover())
	}()
	keyName := ""
	dataDict := make(map[string][]byte)
	tp := tar.NewReader(fp)
	for h, err := tp.Next(); err == nil; h, err = tp.Next() {
		if strings.HasSuffix(h.Name, "/") {
			continue
		}
		f := path.Base(h.Name)
		d := make([]byte, h.Size)
		n, err := tp.Read(d)
		if err != io.EOF || int64(n) != h.Size {
			bailOn(fmt.Errorf("read error: err=%v, n=%d, size=%d",
				err, n, h.Size))
		}
		switch {
		case strings.HasSuffix(f, ".crt"):
			dataDict[f[:len(f)-4]] = d
		case strings.HasSuffix(f, ".key"):
			if keyName != "" {
				bailOn(fmt.Errorf("multiple .key files"))
			}
			keyName = f[:len(f)-4]
			privKey, err = ReadKeyPEMBytes(d)
			bailOn(err)
		default:
			bailOn(fmt.Errorf("extraneous file %s", f))
		}
	}
	if err != io.EOF {
		bailOn(err)
	}
	if keyName == "" {
		bailOn(fmt.Errorf("no .key file"))
	}
	if d, ok := dataDict[keyName]; ok {
		leaf, err = ReadCertPEMBytes(d)
		bailOn(err)
		delete(dataDict, keyName)
	} else {
		bailOn(fmt.Errorf("no %s.crt file", keyName))
	}
	var caCerts []*x509.Certificate
	for _, d := range dataDict {
		cert, err := ReadCertPEMBytes(d)
		bailOn(err)
		caCerts = append(caCerts, cert)
	}
	if len(caCerts) == 0 {
		bailOn(fmt.Errorf("no CA certs"))
	}
	opts = MakeCertPools(caCerts)

	return opts, privKey, leaf, nil
}

// TLSConfigTar : make a (client) tls.Config from the given tar archive file
func TLSConfigTar(tarFile string) (*tls.Config, error) {
	opts, privKey, leaf, err := ReadCertsTar(tarFile)
	if err != nil {
		return nil, err
	}
	return TLSConfig(opts, privKey, leaf)
}

// TLSConfigTarBytes : make a (client) tls.Config from the given tar archive byte slice
func TLSConfigTarBytes(data []byte) (*tls.Config, error) {
	opts, privKey, leaf, err := ReadCertsTarBytes(data)
	if err != nil {
		return nil, err
	}
	return TLSConfig(opts, privKey, leaf)
}

// TLSConfig : make a (client) tls.Config from the given parameters
func TLSConfig(opts *x509.VerifyOptions, privKey interface{}, leaf *x509.Certificate) (*tls.Config, error) {
	chains, err := leaf.Verify(*opts)
	if err != nil {
		return nil, err
	}
	tlsLeaf := tls.Certificate{
		Certificate: [][]byte{leaf.Raw},
		PrivateKey:  privKey,
		Leaf:        leaf,
	}
	tlsPool := x509.NewCertPool()
	for _, c := range chains[0][1:] {
		tlsPool.AddCert(c)
	}

	tlsConfig := &tls.Config{
		RootCAs:          tlsPool,
		CipherSuites:     []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		CurvePreferences: []tls.CurveID{tls.X25519},
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return &tlsLeaf, nil
		},
	}

	return tlsConfig, nil
}
