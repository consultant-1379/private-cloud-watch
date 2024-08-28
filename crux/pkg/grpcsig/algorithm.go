// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	c "github.com/erixzone/crux/pkg/crux"
)

// ecdsaSha256 and ecdsaSha384 can be trivially added here as
// their public key parsing and verification are provided,
// just not yet tested.

var (
	rsaSha1     = &CryptoT{"rsa-sha1", RsaSha1Verify}
	ecdsaSha512 = &CryptoT{"ecdsa-sha512", EcdsaShaVerify}
	eD25519     = &CryptoT{"ed25519", Ed25519Verify}
)

// CryptoT organizes crypto name and Verify function
type CryptoT struct {
	Name   string
	Verify func(key *[]byte, message []byte, signature *[]byte) *c.Err
}

// CryptoFromString - returns the desired public key string and pointer to verification function
func CryptoFromString(name string) (*CryptoT, *c.Err) {
	switch name {
	case "rsa-sha1":
		return rsaSha1, nil
	case "ecdsa-sha512":
		return ecdsaSha512, nil
	case "ed25519":
		return eD25519, nil
	}
	return nil, c.ErrF("Error - Unknown/Unsupported Signature Crypto: %s", name)
}
