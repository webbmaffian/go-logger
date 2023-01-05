package remote

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/transports/remote/auth"
)

type ClientOptions struct {
	Host        string
	Port        int
	Certificate auth.Certificate
	RootCa      auth.Certificate
	PrivateKey  auth.PrivateKey
}

type client struct {
	opt    ClientOptions
	ctx    context.Context
	dialer tls.Dialer
	conn   net.Conn
	time   func() time.Time
}

func NewClient(ctx context.Context, opt ClientOptions) logger.Transport {
	cert := opt.Certificate.TLS(opt.PrivateKey)

	return &client{
		ctx:  ctx,
		opt:  opt,
		time: time.Now,
		dialer: tls.Dialer{
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
		},
	}
}

func (c *client) SetNowFunc(f func() time.Time) {
	c.time = f
	c.dialer.Config.Time = f
}

func (c client) Address() string {
	var b strings.Builder
	b.Grow(len(c.opt.Host) + 5)
	b.WriteString(c.opt.Host)
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(c.opt.Port))
	return b.String()
}

func (c *client) Write(p []byte) (n int, err error) {
	log.Printf("client: writing '%s'...\n", p)
	var timer *time.Timer

loop:
	for {
		if c.conn == nil {
			// c.conn, err = tls.Dial("tcp", c.Address(), c.dialer.Config)
			log.Println("client: connecting")
			c.conn, err = c.dialer.DialContext(c.ctx, "tcp", c.Address())

			if err != nil {
				log.Println("client: failed to connect:", err)
			} else {
				log.Println("client: connected")
			}
		}

		if c.conn != nil {
			c.conn.SetWriteDeadline(c.time().Add(time.Second * 5))
			n, err = c.conn.Write(p)

			if err == nil {
				break
			}

			log.Println("client: failed to write:", err)

			c.conn.Close()
			c.conn = nil
		}

		if timer == nil {
			timer = time.NewTimer(time.Second * 5)
		} else {
			timer.Reset(time.Second * 5)
		}

		select {
		case <-c.ctx.Done():
			break loop
		case <-timer.C:
			continue
		}
	}

	if timer != nil {
		timer.Stop()
	}

	return
}

func (c *client) Close() error {
	return nil
}
