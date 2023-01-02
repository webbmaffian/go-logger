package auth

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"strings"
)

func CreateCsr(privateKey PrivateKey) (csr Csr, err error) {
	hostname, err := os.Hostname()

	if err != nil {
		return
	}

	return x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.PureEd25519,
		Subject: pkix.Name{
			CommonName: hostname,
		},
	}, privateKey.key)
}

const csrBlockType = "CERTIFICATE REQUEST"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidBlockType = errors.New("invalid block type")
)

type Csr []byte

func (c Csr) Encode(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  csrBlockType,
		Bytes: c,
	})
}

func (c *Csr) Decode(b []byte) (err error) {
	p, _ := pem.Decode(b)

	if p == nil {
		return ErrInvalidInput
	}

	if p.Type != csrBlockType {
		return ErrInvalidBlockType
	}

	*c = p.Bytes

	return
}

func (c Csr) String() string {
	var b strings.Builder
	c.Encode(&b)
	return b.String()
}

func (c Csr) Parse() (*x509.CertificateRequest, error) {
	return x509.ParseCertificateRequest(c)
}
