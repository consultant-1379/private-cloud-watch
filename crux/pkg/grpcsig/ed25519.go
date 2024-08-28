// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"encoding/base64"

	"golang.org/x/crypto/ed25519"

	c "github.com/erixzone/crux/pkg/crux"
)

// Ed25519Verify - verifies signature of message using public key provided in
// one-line OpenSSH format
func Ed25519Verify(publicKey *[]byte, message []byte, sig *[]byte) *c.Err {
	pubkey, cryptoName, cerr := DecodeSSHPublicKey(string(*publicKey))
	_ = cryptoName
	if cerr != nil {
		return c.ErrF("Cannot decode ed25519 Public Key: %v", cerr)
	}
	var edpubkey *ed25519.PublicKey
	switch pubkey.(type) {
	case *ed25519.PublicKey:
		edpubkey = pubkey.(*ed25519.PublicKey)
	default:
		return c.ErrF("Public Key is not ed25519")
	}
	sign, err := base64.StdEncoding.DecodeString(string(*sig))
	if err != nil {
		return c.ErrF("Ed25519 cannot decode Base64: %v\n", err)
	}
	if ed25519.Verify(*edpubkey, message, sign) {
		return nil
	}
	return c.ErrF("Ed25519 signature does not match public key")
}
