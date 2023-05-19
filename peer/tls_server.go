package peer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
)

type TlsServer struct {
	opt      TlsServerOptions
	connPool sync.Pool
	listener net.Listener
}

type TlsServerOptions struct {
	Address       string
	PrivateKey    auth.PrivateKey
	Certificate   auth.Certificate
	RootCa        auth.Certificate
	EntryProc     logger.EntryProcessor
	ClientTimeout time.Duration
	Auth          func(ctx context.Context, x509Cert *x509.Certificate) (err error)
	ErrorHandler  func(err error)
	Debug         func(msg string)
	Clock         fastime.Fastime
	NoCopy        bool
}

func (opt *TlsServerOptions) setDefaults(ctx context.Context) {
	if opt.EntryProc == nil {
		opt.EntryProc = entryEchoer{}
	}

	if opt.ClientTimeout <= 0 {
		opt.ClientTimeout = time.Second * 60
	}

	if opt.Clock == nil {
		opt.Clock = fastime.New().StartTimerD(ctx, time.Second)
	}
}

func NewTlsServer(ctx context.Context, opt TlsServerOptions) (s *TlsServer, err error) {
	opt.setDefaults(ctx)

	var listenConfig net.ListenConfig
	netListener, err := listenConfig.Listen(ctx, "tcp", opt.Address)

	if err != nil {
		return
	}

	s = &TlsServer{
		opt: opt,
	}

	s.listener = tls.NewListener(netListener, &tls.Config{
		Time:         s.opt.Clock.Now,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		NextProtos:   []string{protoAck, proto},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    opt.RootCa.X509Pool(),
		Certificates: opt.Certificate.TLSChain(opt.PrivateKey),
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.PeerCertificates == nil || cs.PeerCertificates[0] == nil {
				return logger.ErrInvalidCertificate
			}

			cert := cs.PeerCertificates[0]

			if cert.SerialNumber == nil {
				return logger.ErrInvalidSerialNumber
			}

			// Ensure `SubjectKeyID` contains one or more uint32
			if cert.SubjectKeyId == nil || len(cert.SubjectKeyId)%4 != 0 {
				return logger.ErrInvalidSubjectKeyId
			}

			if s.opt.Auth == nil {
				return nil
			}

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			return s.opt.Auth(ctx, cert)
		},
	})

	go s.acceptConnections(ctx)

	go func() {
		<-ctx.Done()

		if err := s.listener.Close(); err != nil {
			s.error(err)
		}
	}()

	return
}

func (s *TlsServer) acceptConnections(ctx context.Context) {
	for {
		conn, err := s.listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			s.error(err)
			continue
		}

		s.debug("incoming connection")

		if tlsConn, ok := conn.(*tls.Conn); ok {
			go func(tlsConn *tls.Conn) {
				if err := s.handleConnection(ctx, tlsConn); err != nil {
					if err == io.EOF {
						s.debug("disconnected")
					} else {
						s.error(err)
					}
				}
			}(tlsConn)
		} else {
			s.debug("expected TLS connection")
			conn.Close()
		}
	}
}

func (s *TlsServer) handleConnection(ctx context.Context, tlsConn *tls.Conn) (err error) {
	defer tlsConn.Close()

	if err = tlsConn.HandshakeContext(ctx); err != nil {
		return
	}

	conn, err := s.acquireConn(tlsConn)

	if err != nil {
		return
	}

	defer s.releaseConn(conn)

	return conn.listen(ctx)
}

func (s *TlsServer) acquireConn(tlsConn *tls.Conn) (conn *tlsServerConn, err error) {
	state := tlsConn.ConnectionState()

	if state.PeerCertificates == nil {
		return nil, errors.New("missing peer certificates")
	}

	if v := s.connPool.Get(); v != nil {
		conn = v.(*tlsServerConn)
	} else {
		conn = &tlsServerConn{
			entryProc:     s.opt.EntryProc,
			clock:         s.opt.Clock,
			entry:         new(logger.Entry),
			clientTimeout: s.opt.ClientTimeout,
			noCopy:        s.opt.NoCopy,
		}
	}

	cert := state.PeerCertificates[0]
	conn.validBucketIds = cert.SubjectKeyId
	conn.conn = tlsConn
	conn.ack = state.NegotiatedProtocol == protoAck

	return
}

func (s *TlsServer) releaseConn(conn *tlsServerConn) {
	conn.conn.Close()
	conn.conn = nil
	conn.validBucketIds = nil
	s.connPool.Put(conn)
}

func (c *TlsServer) error(err error) {
	if c.opt.ErrorHandler != nil {
		c.opt.ErrorHandler(err)
	}
}

func (c *TlsServer) debug(msg string) {
	if c.opt.Debug != nil {
		c.opt.Debug(msg)
	}
}
