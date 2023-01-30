package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
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

type CertificateType uint8

const (
	Unchanged CertificateType = iota
	Client
	Server
	Root
)

type CertificateOptions struct {
	Subject     pkix.Name
	BucketIds   []uint32
	PublicKey   ed25519.PublicKey
	Expiry      time.Time
	DNSNames    []string
	IPAddresses []net.IP
	Type        CertificateType
}

func (c CertificateOptions) parseCertificateDetails(cert *x509.Certificate) (err error) {
	cert.Subject = mergePkixNames(cert.Subject, c.Subject)

	if c.BucketIds != nil {
		cert.SubjectKeyId = make([]byte, len(c.BucketIds)*4)

		for i := range c.BucketIds {
			binary.BigEndian.PutUint32(cert.SubjectKeyId[i*4:], c.BucketIds[i])
		}
	}

	if c.PublicKey != nil {
		cert.PublicKey = c.PublicKey
	}

	if !c.Expiry.IsZero() {
		cert.NotAfter = c.Expiry
	}

	if c.DNSNames != nil {
		cert.DNSNames = append(cert.DNSNames, c.DNSNames...)
	}

	if c.IPAddresses != nil {
		cert.IPAddresses = append(cert.IPAddresses, c.IPAddresses...)
	}

	if c.Type != Unchanged {
		cert.IsCA = false
		cert.BasicConstraintsValid = false
		cert.MaxPathLenZero = false
		cert.KeyUsage = x509.KeyUsageDigitalSignature
		cert.ExtKeyUsage = nil

		switch c.Type {

		case Client:
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

		case Server:
			cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

		case Root:
			cert.IsCA = true
			cert.BasicConstraintsValid = true
			cert.MaxPathLen = 0
			cert.MaxPathLenZero = true
			cert.KeyUsage |= x509.KeyUsageCertSign
		}
	}

	return
}

type CertificateDetails interface {
	parseCertificateDetails(cert *x509.Certificate) (err error)
}

func CreateCertificate(rootPrivKey PrivateKey, rootCa Certificate, details ...CertificateDetails) (signedCert Certificate, err error) {
	var template *x509.Certificate
	cert, err := baseCertificate()

	if err != nil {
		return
	}

	if rootCa == nil {
		cert.PublicKey = rootPrivKey.Public()
		template = cert
	} else if template, err = rootCa.X509(); err != nil {
		return
	}

	for _, d := range details {
		if err = d.parseCertificateDetails(cert); err != nil {
			return
		}
	}

	if cert.NotAfter.IsZero() {
		cert.NotAfter = time.Now().AddDate(10, 0, 0)
	}

	return x509.CreateCertificate(rand.Reader, cert, template, cert.PublicKey, rootPrivKey.key)
}

func baseCertificate() (cert *x509.Certificate, err error) {
	certId, err := uuid.NewRandom()

	if err != nil {
		return
	}

	var certSerial big.Int
	certSerial.SetBytes(certId[:])

	cert = &x509.Certificate{
		SerialNumber:       &certSerial,
		SignatureAlgorithm: x509.PureEd25519,
		NotBefore:          time.Now(),
		KeyUsage:           x509.KeyUsageDigitalSignature,
	}

	return
}

func CertificateX509(cert *x509.Certificate) Certificate {
	return cert.Raw
}

type Certificate []byte

const certBlockType = "CERTIFICATE"

func (c Certificate) Id() (id uuid.UUID) {
	cert, err := c.X509()

	if err != nil {
		return
	}

	b := cert.SerialNumber.Bytes()

	if len(b) == 16 {
		copy(id[:], b)
	}

	return
}

func (c Certificate) EncodePEM(w io.Writer) (err error) {
	return pem.Encode(w, &pem.Block{
		Type:  certBlockType,
		Bytes: c,
	})
}

func (c *Certificate) DecodePEM(b []byte) (err error) {
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
	c.EncodePEM(&b)
	return b.String()
}

func (c Certificate) PEM() []byte {
	var b bytes.Buffer
	c.EncodePEM(&b)
	return b.Bytes()
}

func (c Certificate) X509() (*x509.Certificate, error) {
	return x509.ParseCertificate(c)
}

func (c Certificate) TLS(key PrivateKey) *tls.Certificate {
	cert, err := c.X509()

	if err != nil {
		log.Println(err)
		return nil
	}

	return &tls.Certificate{
		Certificate:                  [][]byte{c},
		PrivateKey:                   key.key,
		SupportedSignatureAlgorithms: []tls.SignatureScheme{tls.Ed25519},
		Leaf:                         cert,
	}
}

func (c Certificate) TLSChain(key PrivateKey) []tls.Certificate {
	cert, err := c.X509()

	if err != nil {
		log.Println(err)
		return nil
	}

	return []tls.Certificate{
		{
			Certificate:                  [][]byte{c},
			PrivateKey:                   key.key,
			SupportedSignatureAlgorithms: []tls.SignatureScheme{tls.Ed25519},
			Leaf:                         cert,
		},
	}
}

func (c Certificate) X509Pool(certPool ...*x509.CertPool) *x509.CertPool {
	var pool *x509.CertPool

	if certPool != nil && certPool[0] != nil {
		pool = certPool[0]
	} else {
		pool = x509.NewCertPool()
	}

	cert, err := c.X509()

	if err != nil {
		log.Println(err)
		return nil
	}

	pool.AddCert(cert)

	return pool
}

func (c Certificate) ToFile(path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return
	}

	defer f.Close()

	return c.EncodePEM(f)
}

func (c *Certificate) FromFile(path string) (err error) {
	b, err := os.ReadFile(path)

	if err != nil {
		return
	}

	return c.DecodePEM(b)
}

func mergePkixNames(n1 pkix.Name, nn ...pkix.Name) pkix.Name {
	for _, n2 := range nn {
		if n2.CommonName != "" {
			n1.CommonName = n2.CommonName
		}

		if n2.SerialNumber != "" {
			n1.SerialNumber = n2.SerialNumber
		}

		n1.Country = append(n1.Country, n2.Country...)
		n1.ExtraNames = append(n1.ExtraNames, n2.ExtraNames...)
		n1.Locality = append(n1.Locality, n2.Locality...)
		n1.Names = append(n1.Names, n2.Names...)
		n1.Organization = append(n1.Organization, n2.Organization...)
		n1.OrganizationalUnit = append(n1.OrganizationalUnit, n2.OrganizationalUnit...)
		n1.PostalCode = append(n1.PostalCode, n2.PostalCode...)
		n1.Province = append(n1.Province, n2.Province...)
		n1.StreetAddress = append(n1.StreetAddress, n1.StreetAddress...)
	}

	return n1
}
