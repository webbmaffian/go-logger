package peer

import (
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
	"github.com/webbmaffian/go-logger/internal/channel"
	"github.com/webbmaffian/go-logger/logerror"
)

const (
	proto    = "v1.0"
	protoAck = "v1.0-ack"
)

const xidLen = 12

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
	BufferFilepath   string           // Used for the queue buffer of log entries. Default: logs.bin
	BufferSize       int              // Number of entries in the buffer. Default: 100
	ServerAckTimeout time.Duration
	Debug            Debugger
}

func (opt *TlsClientOptions) setDefaults() {
	if opt.BufferFilepath == "" {
		opt.BufferFilepath = "logs.bin"
	}

	if opt.BufferSize <= 0 {
		opt.BufferSize = 100
	}

	if opt.ServerAckTimeout <= 0 {
		opt.ServerAckTimeout = time.Second * 3
	}

	if opt.Debug == nil {
		opt.Debug = nilDebugger{}
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
	c.opt.Debug.Debug("Processing entries...")

	for {
		_, err := c.ch.Wait()

		if err != nil {
			if err != io.EOF {
				c.opt.Debug.Error(err)
			}

			break
		}

		c.ensureConnection(ctx)

		if err := c.ch.ReadToCallback(c.processEntry, true); err != nil && err != io.EOF {
			c.opt.Debug.Error(err)
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

	c.opt.Debug.Info("Sent %d bytes: %v", size, b)

	return nil
}

func (c *TlsClient) processResponses() {
	c.opt.Debug.Debug("Processing responses...")

	var buf [1]byte

	for {
		_, err := c.ch.WaitUntilRead()

		if err != nil {
			c.opt.Debug.Error(err)
			break
		}

		c.opt.Debug.Debug("Wake up - message was sent")

		if !c.ack || c.conn == nil {
			c.opt.Debug.Notice("Skipping: ack %t, conn %t", c.ack, c.conn != nil)
			continue
		}

		c.opt.Debug.Debug("Waiting up to %.2f seconds for acknowledgement...", c.opt.ServerAckTimeout.Seconds())
		c.conn.SetReadDeadline(c.clock.Now().Add(c.opt.ServerAckTimeout))
		n, err := c.conn.Read(buf[:])

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				c.opt.Debug.Notice("No acknowledgement received in time - disconnecting and rewinding.")
			} else if err != io.EOF {
				c.opt.Debug.Notice("Error: %s - disconnecting and rewinding.", err.Error())
			}

			c.disconnect()
			c.ch.Rewind()
			continue
		}

		if n != 0 {
			resp := respType(buf[0])

			switch resp {

			case respAckNOK:
				c.opt.Debug.Notice("Ack received: %s", logerror.ErrInvalidEntry.Error())
				c.ch.Ack()

			case respAckOK:
				c.opt.Debug.Info("Ack reveived!")
				c.ch.Ack()

			}
		}
	}
}

func (c *TlsClient) setupDialer() {
	cert := c.opt.Certificate.TLS(c.opt.PrivateKey)
	c.dialer = tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				c.opt.Debug.Debug("the server is requesting a certificate")
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
			c.opt.Debug.Debug("dialing " + address + " over " + network + "...")
			return nil
		}
	}
}

func (c *TlsClient) ensureConnection(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		c.tryConnect(ctx)
	} else {
		c.opt.Debug.Debug("Ensured connection")
	}
}

func (c *TlsClient) tryConnect(ctx context.Context) {
	if err := c.reconnect(ctx); err != nil {
		c.opt.Debug.Notice("Failed to connect: %s", err.Error())
		c.retryConnect(ctx)
	}
}

func (c *TlsClient) retryConnect(ctx context.Context) {
	c.backoff.Reset()

	for {
		dur := c.backoff.Duration()

		c.opt.Debug.Debug("Retrying to connect in %.2f seconds", dur.Seconds())
		time.Sleep(dur)

		if err := c.connect(ctx); err != nil {
			c.opt.Debug.Notice("Failed to reconnect: %s", err.Error())
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

	c.opt.Debug.Info("Connected!")

	c.setAck(c.conn.ConnectionState().NegotiatedProtocol == protoAck)

	return
}

func (c *TlsClient) disconnect() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c._disconnect()
}

func (c *TlsClient) _disconnect() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
		c.opt.Debug.Debug("Closed connection")
	}

	return
}

func (c *TlsClient) setAck(ack bool) {
	c.ack = ack
}

func (*TlsClient) entrySize(b []byte) int {
	return int(binary.BigEndian.Uint16(b[:2]))
}
