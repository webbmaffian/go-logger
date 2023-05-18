package peer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
	"github.com/webbmaffian/go-mad/channel"
)

const (
	proto    = "v1.0"
	protoAck = "v1.0-ack"
)

const xidLen = 12

type TlsClient struct {
	conn     *tls.Conn
	dialer   tls.Dialer
	ch       *channel.AckByteChannel
	clock    fastime.Fastime
	opt      TlsClientOptions
	backoff  backoff.Backoff
	ack      bool
	ackAwait int
	ackCond  sync.Cond
}

type TlsClientOptions struct {
	Address        string
	PrivateKey     auth.PrivateKey
	Certificate    auth.Certificate
	RootCa         auth.Certificate
	BufferFilepath string
	BufferSize     int
	ErrorHandler   func(err error)
	Debug          func(msg string)
}

func NewTlsClient(ctx context.Context, opt TlsClientOptions) (c *TlsClient, err error) {
	c = &TlsClient{
		clock:   fastime.New().StartTimerD(ctx, time.Second),
		ackCond: sync.Cond{L: &sync.Mutex{}},
		backoff: backoff.Backoff{
			Factor: 2,
			Min:    time.Second,
			Max:    time.Second * 64,
		},
	}

	if c.ch, err = channel.NewAckByteChannel(opt.BufferFilepath, opt.BufferSize, logger.MaxEntrySize); err != nil {
		return
	}

	// We have most likely lost any acknowledgements since restart,
	// so acknowledge any pending entries just in case.
	// TODO: Resend instead, and do this everytime we get disconnected
	// c.ch.AckAll()

	c.setupDialer()
	c.setAck(c.ch.UnackLen() != 0)

	go c.processEntries(ctx)

	go func() {
		<-ctx.Done()
		c.ch.CloseWriting()
	}()

	return
}

func (c *TlsClient) close() error {
	return c.ch.Close()
}

func (c *TlsClient) Send(e *logger.Entry) {
	c.ch.WriteOrFail(func(b []byte) {
		e.Encode(b)
	})
}

func (c *TlsClient) processEntries(ctx context.Context) {
	for {
		b, ok := c.ch.ReadOrBlock()

		if !ok {
			break
		}

		c.ensureConnection(ctx)
		c.processEntry(ctx, b)
	}

	if err := c.close(); err != nil {
		c.error(err)
	}
}

func (c *TlsClient) processEntry(ctx context.Context, b []byte) {
	var bytesWritten int
	size := c.entrySize(b)
	b = b[:size]

	for bytesWritten < size {
		s, err := c.conn.Write(b)
		bytesWritten += s

		if err == nil {
			continue
		}

		if err != io.ErrShortWrite {
			bytesWritten = 0
			c.error(err)
			c.tryConnect(ctx)
		}
	}

	c.expectAck()
}

// Signal that we are expecting an acknowledgement
func (c *TlsClient) expectAck() {
	c.ackCond.L.Lock()
	defer c.ackCond.L.Unlock()

	c.ackAwait++
	c.ackCond.Signal()
}

func (c *TlsClient) awaitAck() {
	c.ackCond.L.Lock()
	defer c.ackCond.L.Unlock()

	for c.ackAwait > 0 {
		c.ackCond.Wait()
		c.ackAwait--
	}
}

func (c *TlsClient) ackEntries(ctx context.Context) {
	var buf [xidLen]byte
	l := 0

	for {
		c.awaitAck()

		for {
			r, err := c.conn.Read(buf[l:])
			l += r

			if l == xidLen {
				c.ch.Ack(func(b []byte) bool {
					return bytes.Equal(c.entryId(b), buf[:])
				})
				l = 0
				break
			}

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}

				c.error(err)
				c.tryConnect(ctx)
			}
		}
	}

}

func (c *TlsClient) setupDialer() {
	cert := c.opt.Certificate.TLS(c.opt.PrivateKey)
	c.dialer = tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				c.opt.Debug("client: the server is requesting a certificate")
				return cert, nil
			},
			RootCAs:            c.opt.RootCa.X509Pool(),
			MinVersion:         tls.VersionTLS13,
			MaxVersion:         tls.VersionTLS13,
			NextProtos:         []string{protoAck, proto},
			ClientSessionCache: tls.NewLRUClientSessionCache(8),
			Time:               c.clock.Now,
		},
		NetDialer: &net.Dialer{
			Timeout: time.Second * 5,
		},
	}

	if c.opt.Debug != nil {
		c.dialer.NetDialer.Control = func(network, address string, _ syscall.RawConn) error {
			c.debug("client: dialing " + address + " over " + network + "...")
			return nil
		}
	}
}

func (c *TlsClient) ensureConnection(ctx context.Context) {
	if c.conn == nil {
		c.tryConnect(ctx)
	}
}

func (c *TlsClient) tryConnect(ctx context.Context) {
	if err := c.reconnect(ctx); err != nil {
		c.error(err)
		c.retryConnect(ctx)
	}
}

func (c *TlsClient) retryConnect(ctx context.Context) {
	c.backoff.Reset()

	for {
		time.Sleep(c.backoff.Duration())

		if err := c.connect(ctx); err != nil {
			c.error(err)
			continue
		}

		break
	}
}

func (c *TlsClient) reconnect(ctx context.Context) (err error) {
	if err = c.disconnect(); err != nil {
		return
	}

	return c.connect(ctx)
}

func (c *TlsClient) connect(ctx context.Context) (err error) {
	var (
		conn net.Conn
		ok   bool
	)

	if conn, err = c.dialer.DialContext(ctx, "tcp", c.opt.Address); err != nil {
		return
	}

	if c.conn, ok = conn.(*tls.Conn); !ok {
		return errors.New("expected TLS connection")
	}

	c.setAck(c.conn.ConnectionState().NegotiatedProtocol == protoAck)

	return
}

func (c *TlsClient) disconnect() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}

	return
}

func (c *TlsClient) setAck(ack bool) {
	c.ack = ack

}

func (c *TlsClient) error(err error) {
	if c.opt.ErrorHandler != nil {
		c.opt.ErrorHandler(err)
	}
}

func (c *TlsClient) debug(msg string) {
	if c.opt.Debug != nil {
		c.opt.Debug(msg)
	}
}

func (*TlsClient) entrySize(b []byte) int {
	return int(binary.BigEndian.Uint16(b[:2]))
}

// XID
func (*TlsClient) entryId(b []byte) []byte {
	return b[6 : 6+xidLen]
}
