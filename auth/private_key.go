package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"strings"
)

func CreatePrivateKey() (privKey PrivateKey, err error) {
	_, privKey.key, err = ed25519.GenerateKey(rand.Reader)
	return
}

func LoadPrivateKey(key ed25519.PrivateKey) PrivateKey {
	return PrivateKey{
		key: key,
	}
}

const privKeyBlockType = "PRIVATE KEY"

type PrivateKey struct {
	key ed25519.PrivateKey
}

func (p PrivateKey) Encode(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  privKeyBlockType,
		Bytes: p.key,
	})
}

func (p *PrivateKey) Decode(b []byte) (err error) {
	block, _ := pem.Decode(b)

	if block == nil {
		return ErrInvalidInput
	}

	if block.Type != privKeyBlockType {
		return ErrInvalidBlockType
	}

	p.key = block.Bytes

	return
}

func (p PrivateKey) String() string {
	var b strings.Builder
	p.Encode(&b)
	return b.String()
}
