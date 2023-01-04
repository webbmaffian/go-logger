package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"os"
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
	cert.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}

	return x509.CreateCertificate(rand.Reader, &cert, &cert, rootPrivKey.key.Public(), rootPrivKey.key)
}

// func CreateServerCertificate()

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
	// cert.IPAddresses = req.IPAddresses

	log.Println("ip addresses:", req.IPAddresses)

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
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	return
}

func CertificateX509(cert *x509.Certificate) Certificate {
	return cert.Raw
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

func (c Certificate) Bytes() []byte {
	var b bytes.Buffer
	c.Encode(&b)
	return b.Bytes()
}

func (c Certificate) Parse() (*x509.Certificate, error) {
	return x509.ParseCertificate(c)
}

func (c Certificate) CertChain(key PrivateKey) []tls.Certificate {
	// log.Println("private key:", string(key.Bytes()))
	// cert, err := tls.X509KeyPair(c.Bytes(), key.Bytes())

	// if err != nil {
	// 	log.Println("ERROR:", err)
	// }

	// return []tls.Certificate{cert}

	cert, _ := c.Parse()

	return []tls.Certificate{
		{
			Certificate:                  [][]byte{c},
			PrivateKey:                   key.key,
			SupportedSignatureAlgorithms: []tls.SignatureScheme{tls.Ed25519},
			Leaf:                         cert,
		},
	}
}

func (c Certificate) CertPool(certPool *x509.CertPool) *x509.CertPool {
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	cert, err := x509.ParseCertificate(c)

	if err != nil {
		log.Println(err)
	}

	certPool.AddCert(cert)

	return certPool
}

func (c Certificate) ToFile(path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return
	}

	defer f.Close()

	return c.Encode(f)
}

func (c Certificate) FromFile(path string) (err error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return
	}

	return c.Decode(b)
}
