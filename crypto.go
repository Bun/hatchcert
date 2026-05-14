package hatchcert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// parseKey decodes a PEM encoded key into its corresponding private key type.
// Primarily used for the account key.
func parseKey(x string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(x))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	if strings.HasPrefix(block.Type, "EC PRIVATE") {
		return x509.ParseECPrivateKey(block.Bytes)
	}
	if strings.HasPrefix(block.Type, "RSA PRIVATE") {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	return x509.ParsePKCS8PrivateKey(block.Bytes)
}

// TODO: probably migrate this to encodePrivateKey
func encodeECKey(key *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func generateECKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func encodePrivateKey(key any) []byte {
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	})
}

func generateKey(keytype string) (crypto.Signer, error) {
	switch keytype {
	case "", "p256", "p-256":
		// Default key type
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "p384", "p-384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "rsa", "rsa2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "rsa3072":
		return rsa.GenerateKey(rand.Reader, 3072)
	case "rsa4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	}
	return nil, fmt.Errorf("unsupported key type %q", keytype)
}
