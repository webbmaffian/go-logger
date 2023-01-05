package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"net"
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
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}, privateKey.key)
}

const csrBlockType = "CERTIFICATE REQUEST"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrInvalidBlockType = errors.New("invalid block type")
)

type Csr []byte

func (c Csr) parseCertificateDetails(cert *x509.Certificate) (err error) {
	req, err := c.Parse()

	if err != nil {
		return
	}

	if req.PublicKeyAlgorithm != x509.Ed25519 || req.SignatureAlgorithm != x509.PureEd25519 {
		return ErrInvalidSignatureAlgorithm
	}

	cert.Subject = mergePkixNames(cert.Subject, req.Subject)
	cert.DNSNames = append(cert.DNSNames, req.DNSNames...)
	cert.IPAddresses = append(cert.IPAddresses, req.IPAddresses...)

	if req.PublicKey != nil {
		cert.PublicKey = req.PublicKey
	}

	return
}

func (c Csr) EncodePEM(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  csrBlockType,
		Bytes: c,
	})
}

func (c *Csr) DecodePEM(b []byte) (err error) {
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
	c.EncodePEM(&b)
	return b.String()
}

func (c Csr) PEM() []byte {
	var b bytes.Buffer
	c.EncodePEM(&b)
	return b.Bytes()
}

func (c Csr) Parse() (*x509.CertificateRequest, error) {
	return x509.ParseCertificateRequest(c)
}

func (c Csr) ToFile(path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return
	}

	defer f.Close()

	return c.EncodePEM(f)
}

func (c Csr) FromFile(path string) (err error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return
	}

	return c.DecodePEM(b)
}
