// Package crypt encrypts the request bodies of the agent with the server's RSA
// public key and decrypts them on the server with the private key.
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// aesKeySize selects AES-256.
const aesKeySize = 32

var (
	errNoPEM    = errors.New("no PEM data found")
	errNotRSA   = errors.New("not an RSA key")
	errTooShort = errors.New("encrypted message is too short")
)

// Encrypt encrypts data with the public key. RSA can encrypt only a few
// hundred bytes, so the data is encrypted with a random AES-256 key in GCM
// mode, and RSA-OAEP encrypts just that key. The result is the encrypted key,
// the nonce and the encrypted data, one after another.
func Encrypt(data []byte, key *rsa.PublicKey) ([]byte, error) {
	// rand.Read never fails since Go 1.24.
	aesKey := make([]byte, aesKeySize)
	rand.Read(aesKey)

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, aesKey, nil)
	if err != nil {
		return nil, err
	}

	gcm, err := newGCM(aesKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	out := append(encryptedKey, nonce...)
	return gcm.Seal(out, nonce, data, nil), nil
}

// Decrypt reverses Encrypt with the private key of the same pair.
func Decrypt(data []byte, key *rsa.PrivateKey) ([]byte, error) {
	size := key.Size()
	if len(data) < size {
		return nil, errTooShort
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), nil, key, data[:size], nil)
	if err != nil {
		return nil, err
	}

	gcm, err := newGCM(aesKey)
	if err != nil {
		return nil, err
	}

	data = data[size:]
	if len(data) < gcm.NonceSize() {
		return nil, errTooShort
	}

	return gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// LoadPublicKey reads an RSA public key from a PEM file in the PKIX format
// ("PUBLIC KEY"), as openssl writes it.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	der, err := loadPEM(path, "PUBLIC KEY")
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errNotRSA
	}
	return rsaKey, nil
}

// LoadPrivateKey reads an RSA private key from a PEM file in the PKCS #8
// format ("PRIVATE KEY"), as openssl writes it.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	der, err := loadPEM(path, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errNotRSA
	}
	return rsaKey, nil
}

// loadPEM returns the contents of the first PEM block in the file. The type
// check turns swapped key files into a clear error.
func loadPEM(path, blockType string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errNoPEM
	}
	if block.Type != blockType {
		return nil, fmt.Errorf("want a %s PEM block, got %s", blockType, block.Type)
	}
	return block.Bytes, nil
}
