package peer

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
	"github.com/webbmaffian/go-logger/logerror"
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
	ch      *channel.AckByteChannel
	clock   fastime.Fastime
	opt     TlsClientOptions
	backoff backoff.Backoff
	mu      sync.Mutex
	ack     bool
}

type TlsClientOptions struct {
	Address          string           // Host and port (e.g. 127.0.0.1:4610) that the client should connect to.
	PrivateKey       auth.PrivateKey  // Private key, used for encryption and authentication.
	Certificate      auth.Certificate // Certificate, used for encryption and authentication.
	RootCa           auth.Certificate // Root certificate authority, used for authenticating the server.
	BufferFilepath   string           // Used for the queue buffer of log entries. Default: logs.bin
	BufferSize       int              // Number of entries in the buffer. Default: 100
	ServerAckTimeout time.Duration
	ErrorHandler     func(err error)  // Callback for non-fatal errors.
	Debug            func(msg string) // Callback for debugging events.
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
		backoff: backoff.Backoff{
			Factor: 2,
			Min:    time.Second,
			Max:    time.Second * 64,
		},
	}

	if c.ch, err = channel.NewAckByteChannel(opt.BufferFilepath, opt.BufferSize, logger.MaxEntrySize, true); err != nil {
		return
	}

	c.setupDialer()

	go c.processEntries(ctx)
	go c.processResponses()

	go func() {
		<-ctx.Done()
		c.ch.CloseWriting()
	}()

	return
}

func (c *TlsClient) WaitUntilSent() error {
	return c.ch.WaitUntilEmpty()
}

// Close the client gracefully. Will block until closed, or the context got cancelled.
func (c *TlsClient) Close(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		c.close()
	}()

	// Ensure that no more entries are written
	c.ch.CloseWriting()

	// Wait until all written entries have been sent, or abort on context cancellation
	if err = c.WaitUntilSent(); err != nil {
		return
	}

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
		_, err := c.ch.Wait()

		if err != nil {
			c.error(err)
			break
		}

		c.ensureConnection(ctx)

		if err := c.ch.ReadToCallback(c.processEntry, true); err != nil && err != channel.ErrEmpty {
			c.error(err)
			c.disconnect()
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

		if err != nil && err != io.EOF {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				c.debug(err.Error())
			} else {
				c.error(err)
			}

			c.disconnect()
			c.ch.Rewind()
			continue
		}

		if n != 0 {
			resp := respType(buf[0])

			switch resp {

			case respAckNOK:
				c.ch.Ack()
				c.error(logerror.ErrInvalidEntry)

			case respAckOK:
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
				c.debug("the server is requesting a certificate")
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
			c.debug("dialing " + address + " over " + network + "...")
			return nil
		}
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
	log.Println("reconnecting")
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

	c.setAck(c.conn.ConnectionState().NegotiatedProtocol == protoAck)

	return
}

func (c *TlsClient) disconnect() (err error) {
	log.Println("disconnecting...")
	c.mu.Lock()
	defer c.mu.Unlock()

	return c._disconnect()
}

func (c *TlsClient) _disconnect() (err error) {
	log.Println("disconnecting")
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
