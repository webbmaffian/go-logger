package logger

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"syscall"
	"time"

	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger/auth"
)

type ClientTLS struct {
	Address     string
	PrivateKey  auth.PrivateKey
	Certificate auth.Certificate
	RootCa      auth.Certificate
	Clock       fastime.Fastime
	dialer      tls.Dialer
}

func (opt *ClientTLS) connect(ctx context.Context) (conn net.Conn, err error) {
	if opt.dialer.Config == nil {
		cert := opt.Certificate.TLS(opt.PrivateKey)
		opt.dialer = tls.Dialer{
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
	}

	return opt.dialer.DialContext(ctx, "tcp", opt.Address)
}

func (opt *ClientTLS) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.Clock.Now().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
