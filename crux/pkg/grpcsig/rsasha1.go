// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"

	c "github.com/erixzone/crux/pkg/crux"
)

// RsaSha1Verify - verifies a message hashed with the OpenSSH
// one-line format encoded publickey is equal to a provided signature (sig)
// signed with the paired private key.
func RsaSha1Verify(publicKey *[]byte, message []byte, signature *[]byte) *c.Err {
	pubkey, cryptoName, cerr := DecodeSSHPublicKey(string(*publicKey))
	if cerr != nil {
		return c.ErrF("Cannot decode rsa-sha1 Public Key: %v", cerr)
	}
	// println("CryptoName: " + cryptoName)
	_ = cryptoName
	var pk *rsa.PublicKey
	switch pubkey.(type) {
	case *rsa.PublicKey:
		pk = pubkey.(*rsa.PublicKey)
	default:
		return c.ErrF("Public Key is not rsa-sha1")
	}

	h := sha1.New()
	if _, err := h.Write(message); err != nil {
		return c.ErrF("Hashing failed in RsaSha1Verify: %v", err)
	}
	sign, err := base64.StdEncoding.DecodeString(string(*signature))
	if err != nil {
		return c.ErrF("Base64 decode failed in RsaSha1Verify: %v\n", err)
	}
	err = rsa.VerifyPKCS1v15(pk, crypto.SHA1, h.Sum(nil), sign)
	if err != nil {
		return c.ErrF("Rsa-sha1 signature does not match public key")
	}
	return nil
}
