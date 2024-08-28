/*
	this is adapted from https://github.com/gtank/cryptopasta
	thanks to George Tankersley for his sample code
	see the Creative Commons CC0 copyright information there

	provides symmetric authenticated encryption using 256-bit AES-GCM with a random nonce.
*/

package flock

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// NonceSize : chosen to match crypt.NewGCM().NonceSize()
const NonceSize = 12

// Nonce : a standard number of random bytes
type Nonce [NonceSize]byte

// NewEncryptionKey generates a random 256-bit key for Encrypt() and Decrypt().
func NewEncryptionKey() (*Key, error) {
	key := Key{}
	_, err := io.ReadFull(rand.Reader, key[:])
	return &key, err
}

// NewGCMNonce generates a random nonce for use with gcm.Seal.
func NewGCMNonce() (*Nonce, error) {
	nonce := Nonce{}
	_, err := io.ReadFull(rand.Reader, nonce[:])
	return &nonce, err
}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Encrypt(plaintext []byte, key *Key) ([]byte, error) {
	return EncryptWithAdd(plaintext, nil, key)
}

// EncryptWithAdd allows additional unencrypted text to be authenticated
func EncryptWithAdd(plaintext, addtext []byte, key *Key) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := NewGCMNonce()
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce[:], nonce[:], plaintext, addtext), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Decrypt(ciphertext []byte, key *Key) ([]byte, error) {
	return DecryptWithAdd(ciphertext, nil, key)
}

// DecryptWithAdd verifies additional text in the authenticated payload
func DecryptWithAdd(ciphertext, addtext []byte, key *Key) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("short ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		addtext,
	)
}
