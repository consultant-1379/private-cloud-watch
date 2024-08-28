// (c) Ericsson AB 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package grpcsig

import (
	"crypto"
	"crypto/ecdsa"
	"encoding/base64"
	"math/big"

	"golang.org/x/crypto/ssh"

	c "github.com/erixzone/crux/pkg/crux"
)

// EcdsaShaVerify - verifies a message hashed with the OpenSSH
// one-line format encoded publickey is equal to a provided signature (sig)
// signed with the paired private key.
func EcdsaShaVerify(publicKey *[]byte, message []byte, sig *[]byte) *c.Err {
	var cerr *c.Err
	// Unpack the openssh-formated public key
	pubkey, cryptoName, cerr := DecodeSSHPublicKey(string(*publicKey))
	if cerr != nil {
		return c.ErrF("Cannot decode ecdsa Public Key: %v", cerr)
	}
	// println("CryptoName: " + cryptoName)
	_ = cryptoName
	// Cast to the ecdsa PublicKey type
	var ecpubkey *ecdsa.PublicKey
	switch pubkey.(type) {
	case *ecdsa.PublicKey:
		ecpubkey = pubkey.(*ecdsa.PublicKey)
	default:
		return c.ErrF("Public Key is not ecdsa")
	}
	var ecHash crypto.Hash
	bits := ecpubkey.Curve.Params().BitSize
	switch {
	case bits <= 256:
		ecHash = crypto.SHA256
	case bits <= 384:
		ecHash = crypto.SHA384
	case bits <= 521:
		ecHash = crypto.SHA512
	}
	// Hash the message
	h := ecHash.New()
	h.Write(message)
	digest := h.Sum(nil)
	// Decode and unmarshall the R,S style ecdsa signature
	sign, err := base64.StdEncoding.DecodeString(string(*sig))
	if err != nil {
		return c.ErrF("Base64 Decode Error: %v\n", err)
	}
	var ecdsaSig EcdsaSigT
	err = ssh.Unmarshal(sign, &ecdsaSig)
	if err != nil {
		return c.ErrF("Cannot unmarshal ecdsa signature: %v", err)
	}
	// Verify with the library function
	if ecdsa.Verify((*ecdsa.PublicKey)(ecpubkey), digest, ecdsaSig.R, ecdsaSig.S) {
		return nil
	}
	return c.ErrF("Ecdsa signature does not match public key")
}

// EcdsaSigT  - is the exposed signature data
type EcdsaSigT struct {
	R *big.Int
	S *big.Int
}

// EcdsaSignature - unmarshalls the R,S form of an ECDSA signature from ssh-agent, figures out
// the hash size, returns a SigPairT
func EcdsaSignature(sigBlob []byte) (SigPairT, *c.Err) {
	var ecdsaSig EcdsaSigT
	sigpair := SigPairT{}
	err := ssh.Unmarshal(sigBlob, &ecdsaSig) // this is missing the asn.1.lead bytes
	if err != nil {
		return sigpair, c.ErrF("Cannot unmarshal ecdsa signature: %v", err)
	}
	rsize := len(ecdsaSig.R.Bytes())
	hashedwith := ""
	switch rsize {
	case 31, 32:
		hashedwith = "ecdsa-sha256"
	case 65, 66:
		hashedwith = "ecdsa-sha512"
	default:
		return sigpair, c.ErrF("Ecdsa key lengh %d not supported", rsize)
	}
	sigpair.HashAlg = hashedwith
	sigpair.Base64Sig = base64.StdEncoding.EncodeToString(sigBlob)
	return sigpair, nil
}
