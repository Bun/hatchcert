package hatchcert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name    string
		keytype string
		wantErr bool
	}{
		{"Default P256", "", false},
		{"P256 with dash", "p-256", false},
		{"P384", "p384", false},
		{"RSA 2048", "rsa", false},
		{"RSA 4096", "rsa4096", false},
		{"Unsupported", "rsaOneMillion", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateKey(tt.keytype)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("generateKey() returned nil signer without error")
			}
		})
	}
}

func TestEncodePrivateKey(t *testing.T) {
	// Setup keys
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecdsaKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	tests := []struct {
		name     string
		key      any
		wantType string
	}{
		{
			name:     "Encode RSA",
			key:      rsaKey,
			wantType: "PRIVATE KEY",
		},
		{
			name:     "Encode ECDSA",
			key:      ecdsaKey,
			wantType: "PRIVATE KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodePrivateKey(tt.key)

			block, _ := pem.Decode(encoded)
			if block == nil {
				t.Fatal("failed to decode PEM block")
			}

			if block.Type != tt.wantType {
				t.Errorf("got block type %q, want %q", block.Type, tt.wantType)
			}

			// Verify the bytes are actually parseable back to keys
			_, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				t.Errorf("encoded bytes are not valid %s: %v", block.Type, err)
			}
		})
	}
}
