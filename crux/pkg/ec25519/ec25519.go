package ec25519

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"golang.org/x/crypto/curve25519"
	"io"
)

// Keylen : length of curve25519 key
const (
	Keylen int = 32
)

// Key : type of curve25519 key
type Key [Keylen]byte

// ScalarBaseMult : as in base package
func ScalarBaseMult(dest, s *Key) {
	curve25519.ScalarBaseMult((*[Keylen]byte)(dest), (*[Keylen]byte)(s))
}

// ScalarMult : as in base package
func ScalarMult(dest, s, n *Key) {
	curve25519.ScalarMult((*[Keylen]byte)(dest), (*[Keylen]byte)(s), (*[Keylen]byte)(n))
}

// Clamp : ensure a valid secret key, function name per DJB
func Clamp(x *Key) {
	x[0] &= 0xf8  // clear lowest-order 3 bits
	x[31] &= 0x7f // clear highest-order bit
	x[31] |= 0x40 // set 2nd-highest-order bit
}

// NewSecret : generate a random secret key
func NewSecret(secr *Key) (err error) {
	buf := make([]byte, Keylen)
	if _, err = io.ReadFull(rand.Reader, buf); err == nil {
		copy(secr[:], buf)
		Clamp(secr)
	}
	return err
}

// NewKeyPair : generate a random (secret, public) key pair
func NewKeyPair(secr, pub *Key) (err error) {
	if err = NewSecret(secr); err == nil {
		ScalarBaseMult(pub, secr)
	}
	return err
}

// NewKeyPairFromSeed : generate a (secret, public) key pair from an ED25519 seed
func NewKeyPairFromSeed(seed []byte, secr, pub *Key) (err error) {
	if len(seed) < Keylen {
		return fmt.Errorf("ec25519: bad seed length: need at least %d, got %d,", Keylen, len(seed))
	}
	digest := sha512.Sum512(seed[:Keylen])
	copy(secr[:], digest[:])
	Clamp(secr)
	ScalarBaseMult(pub, secr)
	return nil
}

// SharedKey : compute a usable shared key
func SharedKey(secrA, pubB *Key, nonceA, nonceB []byte) []byte {
	var ecshareAB Key

	ScalarMult(&ecshareAB, secrA, pubB)
	h := sha256.New()
	h.Write(ecshareAB[:])
	h.Write(nonceA)
	h.Write(nonceB)
	return h.Sum(nil)
}
