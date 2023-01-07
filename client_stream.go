package logger

import (
	"crypto/tls"
	"log"
	"net"
	"syscall"
	"time"

	"github.com/webbmaffian/go-logger/auth"
)

type ClientTCP struct {
	Address string
	Clock   func() time.Time
}

func (opt ClientTCP) dialer(c *client) func() (net.Conn, error) {
	var dialer net.Dialer

	return func() (net.Conn, error) {
		return dialer.DialContext(c.ctx, "tcp", opt.Address)
	}
}

func (opt ClientTCP) clock() func() time.Time {
	return opt.Clock
}

type ClientTLS struct {
	Address     string
	PrivateKey  auth.PrivateKey
	Certificate auth.Certificate
	RootCa      auth.Certificate
	Clock       func() time.Time
}

func (opt ClientTLS) dialer(c *client) func() (net.Conn, error) {
	cert := opt.Certificate.TLS(opt.PrivateKey)
	dialer := tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				log.Println("client: the server is requesting a certificate")
				return cert, nil
			},
			RootCAs:            opt.RootCa.X509Pool(),
			MinVersion:         tls.VersionTLS13,
			MaxVersion:         tls.VersionTLS13,
			NextProtos:         []string{"wallaaa"},
			ClientSessionCache: tls.NewLRUClientSessionCache(8),
		},
		NetDialer: &net.Dialer{
			Timeout: time.Second * 5,
			Control: func(network, address string, c syscall.RawConn) error {
				log.Println("client: dialing", address, "over", network, "...")
				return nil
			},
		},
	}

	return func() (net.Conn, error) {
		return dialer.DialContext(c.ctx, "tcp", opt.Address)
	}
}

func (opt ClientTLS) clock() func() time.Time {
	return opt.Clock
}
