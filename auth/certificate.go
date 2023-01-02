package auth

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidSignatureAlgorithm = errors.New("invalid signature algorithm - must be ED25519")
)

func CreateRootCA(subject pkix.Name, rootPrivKey PrivateKey, expiry time.Time) (rootCa Certificate, err error) {
	cert, err := baseCertificate(expiry)

	if err != nil {
		return
	}

	cert.IsCA = true
	cert.KeyUsage |= x509.KeyUsageCertSign

	return x509.CreateCertificate(rand.Reader, &cert, &cert, rootPrivKey.key.Public(), rootPrivKey.key)
}

func CreateCertificate(clientId uuid.UUID, csr Csr, rootCa Certificate, rootPrivKey PrivateKey, expiry time.Time) (signedCert Certificate, err error) {
	req, err := csr.Parse()

	if err != nil {
		return
	}

	template, err := rootCa.Parse()

	if err != nil {
		return
	}

	cert, err := baseCertificate(expiry)

	if err != nil {
		return
	}

	if req.SignatureAlgorithm != cert.SignatureAlgorithm {
		err = ErrInvalidSignatureAlgorithm
		return
	}

	cert.Subject = req.Subject
	cert.SubjectKeyId = clientId[:]

	return x509.CreateCertificate(rand.Reader, &cert, template, req.PublicKey, rootPrivKey.key)
}

func baseCertificate(expiry time.Time) (cert x509.Certificate, err error) {
	certId, err := uuid.NewRandom()

	if err != nil {
		return
	}

	var certSerial big.Int
	certSerial.SetBytes(certId[:])

	cert = x509.Certificate{
		SerialNumber:          &certSerial,
		SignatureAlgorithm:    x509.PureEd25519,
		NotBefore:             time.Now(),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	return
}

type Certificate []byte

const certBlockType = "CERTIFICATE"

func (c Certificate) Encode(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  certBlockType,
		Bytes: c,
	})
}

func (c *Certificate) Decode(b []byte) (err error) {
	p, _ := pem.Decode(b)

	if p == nil {
		return ErrInvalidInput
	}

	if p.Type != certBlockType {
		return ErrInvalidBlockType
	}

	*c = p.Bytes

	return
}

func (c Certificate) String() string {
	var b strings.Builder
	c.Encode(&b)
	return b.String()
}

func (c Certificate) Parse() (*x509.Certificate, error) {
	return x509.ParseCertificate(c)
}
