package peer

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
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

var _ logger.Client = (*TlsClient)(nil)

type TlsClient struct {
	conn    *tls.Conn
	dialer  tls.Dialer
	ch      *channel.ByteChannel
	clock   fastime.Fastime
	opt     TlsClientOptions
	backoff backoff.Backoff
	ack     bool
	cond    sync.Cond
}

type TlsClientOptions struct {
	Address        string           // Host and port (e.g. 127.0.0.1:4610) that the client should connect to.
	PrivateKey     auth.PrivateKey  // Private key, used for encryption and authentication.
	Certificate    auth.Certificate // Certificate, used for encryption and authentication.
	RootCa         auth.Certificate // Root certificate authority, used for authenticating the server.
	BufferFilepath string           // Used for the queue buffer of log entries. Default: logs.bin
	BufferSize     int              // Number of entries in the buffer. Default: 100
	ErrorHandler   func(err error)  // Callback for non-fatal errors.
	Debug          func(msg string) // Callback for debugging events.
}

func (opt *TlsClientOptions) setDefaults() {
	if opt.BufferFilepath == "" {
		opt.BufferFilepath = "logs.bin"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 100
	}
}

func NewTlsClient(ctx context.Context, opt TlsClientOptions) (c *TlsClient, err error) {
	opt.setDefaults()

	if err = opt.Certificate.Validate(opt.PrivateKey); err != nil {
		return
	}

	if opt.Certificate.Type() != auth.Client {
		err = errors.New("not a client certificate")
		return
	}

	c = &TlsClient{
		opt:   opt,
		clock: fastime.New().StartTimerD(ctx, time.Second),
		cond:  sync.Cond{L: &sync.Mutex{}},
		backoff: backoff.Backoff{
			Factor: 2,
			Min:    time.Second,
			Max:    time.Second * 64,
		},
	}

	if c.ch, err = channel.NewByteChannel(opt.BufferFilepath, opt.BufferSize, logger.MaxEntrySize, true); err != nil {
		return
	}

	c.setupDialer()

	go c.processEntries(ctx)

	go func() {
		<-ctx.Done()
		c.ch.CloseWriting()
	}()

	return
}

func (c *TlsClient) WaitUntilSent(ctx context.Context) error {
	return c.ch.WaitForSync(ctx)
}

// Close the client gracefully. Will block until closed, or the context got cancelled.
func (c *TlsClient) Close(ctx context.Context) (err error) {

	// Ensure that no more entries are written
	c.ch.CloseWriting()

	// Wait until all written entries have been sent, or abort on context cancellation
	if err = c.WaitUntilSent(ctx); err != nil {
		return
	}

	// Close channel (and memory-mapped file)
	c.close()

	return c.conn.Close()
}

func (c *TlsClient) close() error {
	return c.ch.Close()
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
		ok := c.ch.Wait()

		if !ok {
			break
		}

		c.ensureConnection(ctx)

		if err := c.ch.ReadToCallback(c.processEntry, true); err != nil && err != channel.ErrEmpty {
			c.error(err)
			c.tryConnect(ctx)
		}
	}

	if err := c.close(); err != nil {
		c.error(err)
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

func (c *TlsClient) setupDialer() {
	cert := c.opt.Certificate.TLS(c.opt.PrivateKey)
	c.dialer = tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				c.debug("the server is requesting a certificate")
				return cert, nil
			},
			RootCAs:            c.opt.RootCa.X509Pool(),
			MinVersion:         tls.VersionTLS13,
			MaxVersion:         tls.VersionTLS13,
			NextProtos:         []string{proto},
			ClientSessionCache: tls.NewLRUClientSessionCache(8),
			Time:               c.clock.Now,
		},
		NetDialer: &net.Dialer{
			Timeout: time.Second * 5,
		},
	}

	if c.opt.Debug != nil {
		c.dialer.NetDialer.Control = func(network, address string, _ syscall.RawConn) error {
			c.debug("dialing " + address + " over " + network + "...")
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
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	c.ack = ack
	c.cond.Signal()
}

func (c *TlsClient) error(err error) {
	if c.opt.ErrorHandler != nil {
		c.opt.ErrorHandler(err)
	}
}

func (c *TlsClient) debug(msg string, args ...any) {
	if c.opt.Debug != nil {
		if args == nil {
			c.opt.Debug(msg)
		} else {
			c.opt.Debug(fmt.Sprintf(msg, args...))
		}
	}
}

func (*TlsClient) entrySize(b []byte) int {
	return int(binary.BigEndian.Uint16(b[:2]))
}
