package hatchcert

import (
	"crypto"

	"github.com/go-acme/lego/v3/certcrypto"
)

var DefaultKeyType = certcrypto.EC384

func generatePrivateKey(keyType certcrypto.KeyType) (crypto.PrivateKey, string, error) {
	pk, err := certcrypto.GeneratePrivateKey(keyType)
	if err != nil {
		return nil, "", err
	}

	buf := certcrypto.PEMEncode(pk)
	return pk, string(buf), nil
}

func parseKey(x string) (crypto.PrivateKey, error) {
	return certcrypto.ParsePEMPrivateKey([]byte(x))
}
