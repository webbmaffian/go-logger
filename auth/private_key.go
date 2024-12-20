package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
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

func (p PrivateKey) Public() ed25519.PublicKey {
	k := p.key.Public()
	return k.(ed25519.PublicKey)
}

func (p PrivateKey) EncodePEM(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  privKeyBlockType,
		Bytes: p.DER(),
	})
}

func (p *PrivateKey) DecodePEM(b []byte) (err error) {
	block, _ := pem.Decode(b)

	if block == nil {
		return ErrInvalidInput
	}

	if block.Type != privKeyBlockType {
		return ErrInvalidBlockType
	}

	var privKey any

	if privKey, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		return
	}

	var ok bool

	if p.key, ok = privKey.(ed25519.PrivateKey); !ok {
		return ErrInvalidInput
	}

	return
}

func (p PrivateKey) String() string {
	var b strings.Builder
	p.EncodePEM(&b)
	return b.String()
}

func (p PrivateKey) DER() (der []byte) {
	der, _ = x509.MarshalPKCS8PrivateKey(p.key)
	return
}

func (p PrivateKey) PEM() []byte {
	var b bytes.Buffer
	p.EncodePEM(&b)
	return b.Bytes()
}

func (p PrivateKey) ToFile(path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return
	}

	defer f.Close()

	return p.EncodePEM(f)
}

func (p *PrivateKey) FromFile(path string) (err error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return
	}

	return p.DecodePEM(b)
}
