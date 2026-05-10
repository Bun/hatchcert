package hatchcert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

// TODO: allow key type selection; right now we support P-256 for maximum
// compatibility.

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

func encodeECKey(key *ecdsa.PrivateKey) string {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}))
}

func generateECKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
