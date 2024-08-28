// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"

	c "github.com/erixzone/crux/pkg/crux"
)

// OpenSSH one-line format has 3 space delimited text fields: { key_type, data, comment }
// data is base64 encoded binary which consists of Asn.1 tuples of length (4 bytes) encoded data.
// For RSA keys, there should be three tuples which should be: { key_type, public_exponent, modulus }
// ECDSA keys are more complex, while Ed25519 keys revert to the simpler form.

// FingerprintSSHPublicKey gets a fingerprint from a string representing a one-line public key as written by ssh-keygen
func FingerprintSSHPublicKey(str string) (string, *c.Err) {
	tokens := strings.Split(str, " ")
	if len(tokens) < 2 {
		return "", c.ErrF("invalid public key format string, must contain at least two fields (keytype data [comment])")
	}
	data, err := base64.StdEncoding.DecodeString(tokens[1])
	if err != nil {
		return "", c.ErrF("unable to Base64 decode public key - %s", err.Error())
	}
	md5hash := md5.New()
	md5hash.Write(data)
	rawPrint := fmt.Sprintf("%x", md5hash.Sum(nil))
	fpFormatted := ""
	for i := 0; i < len(rawPrint); i = i + 2 {
		fpFormatted = fmt.Sprintf("%s%s:", fpFormatted, rawPrint[i:i+2])
	}
	fp := strings.TrimSuffix(fpFormatted, ":")
	return fp, nil
}

// DecodeSSHPublicKey - takes an OpenSSH formated public key (one line, 3 space-delimited strings)
// and returns an interface to the parsed .PublicKey
func DecodeSSHPublicKey(str string) (interface{}, string, *c.Err) {
	// split ssh format one-liner into component parts
	tokens := strings.Split(str, " ")
	if len(tokens) < 2 {
		return nil, "", c.ErrF("invalid public key format string, must contain at least two fields (keytype data [comment])")
	}
	// Extract key type, data
	keytype := tokens[0]
	data, err := base64.StdEncoding.DecodeString(tokens[1])
	if err != nil {
		return nil, "", c.ErrF("unable to Base64 decode - %s", err.Error())
	}
	alg, cerr := SignatureFormatToAlg(keytype)
	if cerr != nil {
		return nil, "", cerr
	}
	switch alg {
	case "rsa":
		return ExtractRsaPubkey(data, keytype)
	case "ecdsa":
		return ExtractECDSAPubKey(data, keytype)
	case "ed25519":
		return ExtractEd25519PubKey(data, keytype)
	}
	return nil, "", c.ErrF("unsupported public key format %s", alg)
}

// readLength - reads length encoded data
func readLength(data []byte) ([]byte, uint32, *c.Err) {
	if len(data) < 5 {
		return nil, 0, c.ErrF("readLength -  no data left to read.")
	}
	lbuf := data[0:4]
	buf := bytes.NewBuffer(lbuf)
	var length uint32
	err := binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return nil, 0, c.ErrF("%s", err.Error())
	}
	return data[4:], length, nil
}

// ExtractRsaPubkey - extracts rsa-sha1 formatted OpenSSH public key, provides structured .PublicKey
func ExtractRsaPubkey(data []byte, keytype string) (*rsa.PublicKey, string, *c.Err) {
	data, length, cerr := readLength(data)
	if cerr != nil {
		return nil, "", c.ErrF("unable to read RSA public key format: %v", cerr)
	}
	cryptoName := string(data[0:length])
	data = data[length:]
	var w struct {
		E    *big.Int
		N    *big.Int
		Rest []byte `ssh:"rest"`
	}
	err := ssh.Unmarshal(data, &w)
	if err != nil {
		return nil, cryptoName, c.ErrF("unable to read rsa-sha1 public key format: %s", err.Error())
	}
	if w.E.BitLen() > 24 {
		return nil, cryptoName, c.ErrF("rsa-sha1 public key exponent too large")
	}
	e := w.E.Int64()
	if e < 3 || e&1 == 0 {
		return nil, cryptoName, c.ErrF("rsa-sha1 public key exponent error")
	}
	pubKey := &rsa.PublicKey{
		N: w.N,
		E: int(w.E.Int64()),
	}
	return pubKey, cryptoName, nil
}

// ExtractEd25519PubKey - extracts ed25519 formatted OpenSSH public key, provides structured .PublicKey
func ExtractEd25519PubKey(data []byte, keytype string) (*ed25519.PublicKey, string, *c.Err) {
	data, length, err := readLength(data)
	if err != nil {
		return nil, "", c.ErrF("unable to read ed25519 public key format ")
	}
	cryptoName := string(data[0:length])
	data = data[length:]
	var w struct {
		KeyBytes []byte
		Rest     []byte `ssh:"rest"`
	}
	if err := ssh.Unmarshal(data, &w); err != nil {
		return nil, cryptoName, c.ErrF("%s", err.Error())
	}
	key := ed25519.PublicKey(w.KeyBytes)
	return (*ed25519.PublicKey)(&key), cryptoName, nil
}

// ExtractECDSAPubKey - extracts ecdsa formatted OpenSSH public key, provides structured .PublicKey
func ExtractECDSAPubKey(data []byte, keytype string) (*ecdsa.PublicKey, string, *c.Err) {
	data, length, cerr := readLength(data)
	if cerr != nil {
		return nil, "", c.ErrF("unable to read ecdsa public key format: %v ", cerr)
	}
	cryptoName := string(data[0:length])
	data = data[length:]
	var w struct {
		Curve    string
		KeyBytes []byte
		Rest     []byte `ssh:"rest"`
	}
	err := ssh.Unmarshal(data, &w)
	if err != nil {
		return nil, "", c.ErrF("%s", err.Error())
	}
	key := new(ecdsa.PublicKey)
	switch w.Curve {
	case "nistp256":
		key.Curve = elliptic.P256()
	case "nistp384":
		key.Curve = elliptic.P384()
	case "nistp521":
		key.Curve = elliptic.P521()
	default:
		return nil, cryptoName, c.ErrF("unsupported ecdsa curve")
	}
	key.X, key.Y = elliptic.Unmarshal(key.Curve, w.KeyBytes)
	if key.X == nil || key.Y == nil {
		return nil, cryptoName, c.ErrF("invalid ecdsa curve point")
	}
	return (*ecdsa.PublicKey)(key), cryptoName, nil
}
