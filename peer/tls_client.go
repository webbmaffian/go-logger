package peer

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
	"github.com/webbmaffian/go-logger/internal/channel"
)

const (
	protoV10    = "v1.0"
	protoV10Ack = "v1.0-ack"
	protoV11Ack = "v1.1-ack"
)

var _ logger.Client = (*TlsClient)(nil)

type TlsClient struct {
	ctxCancel context.CancelFunc
	conn      *tls.Conn
	dialer    tls.Dialer
	ch        *channel.ByteChannel
	clock     fastime.Fastime
	opt       TlsClientOptions
	backoff   backoff.Backoff
	mu        sync.Mutex
	ack       bool
}

type TlsClientOptions struct {
	Address          string           // Host and port (e.g. 127.0.0.1:4610) that the client should connect to.
	PrivateKey       auth.PrivateKey  // Private key, used for encryption and authentication.
	Certificate      auth.Certificate // Certificate, used for encryption and authentication.
	RootCa           auth.Certificate // Root certificate authority, used for authenticating the server.
	BufferSize       int              // Number of entries in the buffer. Default: 100
	ServerAckTimeout time.Duration
	ErrorHandler     func(error)
}

func (opt *TlsClientOptions) setDefaults() {
	if opt.BufferSize <= 0 {
		opt.BufferSize = 128
	}

	if opt.ServerAckTimeout <= 0 {
		opt.ServerAckTimeout = time.Second * 3
	}

	if opt.ErrorHandler == nil {
		opt.ErrorHandler = func(_ error) {}
	}
}

func NewTlsClient(opt TlsClientOptions) (c *TlsClient, err error) {
	opt.setDefaults()

	if err = opt.Certificate.Validate(opt.PrivateKey); err != nil {
		return
	}

	if opt.Certificate.Type() != auth.Client {
		err = errors.New("not a client certificate")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	c = &TlsClient{
		ctxCancel: cancel,
		ch:        channel.NewByteChannel(opt.BufferSize, logger.MaxEntrySize),
		opt:       opt,
		clock:     fastime.New().StartTimerD(ctx, time.Second),
		backoff: backoff.Backoff{
			Factor: 2,
			Min:    time.Second,
			Max:    time.Second * 64,
		},
	}

	c.setupDialer()

	go c.processEntries(ctx)
	go c.processResponses()

	return
}

func (c *TlsClient) WaitUntilSent() error {
	return c.ch.WaitUntilEmpty()
}

// Close the client gracefully. Will block until closed, or the context got cancelled.
func (c *TlsClient) CloseWithContext(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		c.Close()
	}()

	// Ensure that no more entries are written
	c.ch.CloseWriting()

	// Wait until all written entries have been sent
	return c.WaitUntilSent()
}

func (c *TlsClient) CloseGracefully(timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return c.CloseWithContext(ctx)
}

// Closes forcefully
func (c *TlsClient) Close() error {
	c.ctxCancel()
	c.ch.Close()
	return c._disconnect()
}

func (c *TlsClient) Now() time.Time {
	return c.clock.Now()
}

func (c *TlsClient) ProcessEntry(_ context.Context, e *logger.Entry) (err error) {
	c.ch.WriteOrReplace(func(b []byte) {
		e.Encode(b)
	})

	return
}

func (c *TlsClient) processEntries(ctx context.Context) {
	for {
		_, err := c.ch.Wait()

		if err != nil {
			if err != io.EOF {
				c.error(err)
			}

			break
		}

		c.ensureConnection(ctx)

		if err := c.ch.ReadToCallback(c.processEntry, true); err != nil && err != io.EOF {
			c.disconnect()
		}
	}
}

func (c *TlsClient) processEntry(b []byte) error {
	var bytesWritten int
	size := c.entrySize(b)
	b = b[:size]

	for bytesWritten < size {
		s, err := c.conn.Write(b[bytesWritten:])
		bytesWritten += s

		if err == nil {
			continue
		}

		if err != io.ErrShortWrite {
			return err
		}
	}

	return nil
}

func (c *TlsClient) processResponses() {
	var buf [1]byte

	for {
		_, err := c.ch.WaitUntilRead()

		if err != nil {
			c.error(err)
			break
		}

		if !c.ack || c.conn == nil {
			continue
		}

		c.conn.SetReadDeadline(c.clock.Now().Add(c.opt.ServerAckTimeout))
		n, err := c.conn.Read(buf[:])

		if err != nil {
			c.disconnect()
			c.ch.Rewind()
			continue
		}

		if n != 0 {
			c.ch.Ack()
		}
	}
}

func (c *TlsClient) setupDialer() {
	cert := c.opt.Certificate.TLS(c.opt.PrivateKey)
	c.dialer = tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return cert, nil
			},
			RootCAs:            c.opt.RootCa.X509Pool(),
			MinVersion:         tls.VersionTLS13,
			MaxVersion:         tls.VersionTLS13,
			NextProtos:         []string{protoV11Ack, protoV10},
			ClientSessionCache: tls.NewLRUClientSessionCache(8),
			Time:               c.clock.Now,
		},
		NetDialer: &net.Dialer{
			Timeout: time.Second * 5,
		},
	}
}

func (c *TlsClient) ensureConnection(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		c.tryConnect(ctx)
	}
}

func (c *TlsClient) tryConnect(ctx context.Context) {
	if err := c.reconnect(ctx); err != nil {
		c.retryConnect(ctx)
	}
}

func (c *TlsClient) retryConnect(ctx context.Context) {
	c.backoff.Reset()

	for {
		time.Sleep(c.backoff.Duration())

		if err := c.connect(ctx); err != nil {
			continue
		}

		break
	}
}

func (c *TlsClient) reconnect(ctx context.Context) (err error) {
	if err = c._disconnect(); err != nil {
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

	c.ack = c.conn.ConnectionState().NegotiatedProtocol == protoV11Ack

	return
}

func (c *TlsClient) disconnect() (err error) {
	return c._disconnect()
}

func (c *TlsClient) _disconnect() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}

	return
}

func (*TlsClient) entrySize(b []byte) int {
	return int(binary.BigEndian.Uint16(b[:2]))
}

func (c *TlsClient) error(err error) {
	c.opt.ErrorHandler(err)
}
