package logger

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"syscall"
	"time"

	"github.com/webbmaffian/go-logger/auth"
)

type ClientTCP struct {
	Address string
	TimeNow func() time.Time
	dialer  net.Dialer
	conn    *net.TCPConn
}

func (opt *ClientTCP) write(ctx context.Context, b []byte) (err error) {
	if opt.conn == nil {
		var conn net.Conn

		if conn, err = opt.dialer.DialContext(ctx, "tcp", opt.Address); err == nil {
			opt.conn = conn.(*net.TCPConn)
		}
	}

	if opt.conn != nil {
		opt.conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
		_, err = opt.conn.Write(b)
	}

	return
}

func (opt *ClientTCP) close() (err error) {
	if opt.conn != nil {
		err = opt.conn.Close()
		opt.conn = nil
	}

	return
}

type ClientTLS struct {
	Address     string
	PrivateKey  auth.PrivateKey
	Certificate auth.Certificate
	RootCa      auth.Certificate
	TimeNow     func() time.Time
	dialer      tls.Dialer
	conn        *tls.Conn
}

func (opt *ClientTLS) write(ctx context.Context, b []byte) (err error) {
	if opt.conn == nil {
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

		var conn net.Conn

		if conn, err = opt.dialer.DialContext(ctx, "tcp", opt.Address); err == nil {
			opt.conn = conn.(*tls.Conn)
		}
	}

	if opt.conn != nil {
		opt.conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
		_, err = opt.conn.Write(b)
	}

	return
}

func (opt *ClientTLS) close() (err error) {
	if opt.conn != nil {
		err = opt.conn.Close()
		opt.conn = nil
	}

	return
}
