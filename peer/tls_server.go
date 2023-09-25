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

	"github.com/google/uuid"
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
	Debug         Debugger
	Clock         fastime.Fastime
	Log           *logger.Logger
	NoCopy        bool
}

func (opt *TlsServerOptions) setDefaults(ctx context.Context) {
	if opt.EntryProc == nil {
		opt.EntryProc = entryEchoer{}
	}

	if opt.ClientTimeout <= 0 {
		opt.ClientTimeout = time.Minute
	}

	if opt.Clock == nil {
		opt.Clock = fastime.New().StartTimerD(ctx, time.Second)
	}

	if opt.Log == nil {
		pool, _ := logger.NewPool(logger.NewDummyWriter(ctx))
		opt.Log = pool.Logger()
	}

	if opt.Debug == nil {
		opt.Debug = nilDebugger{}
	}
}

func NewTlsServer(ctx context.Context, opt TlsServerOptions) (s *TlsServer, err error) {
	opt.setDefaults(ctx)

	if err = opt.Certificate.Validate(opt.PrivateKey); err != nil {
		return
	}

	if opt.Certificate.Type() != auth.Server {
		err = errors.New("not a server certificate")
		return
	}

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

			if cert.SerialNumber == nil || cert.SerialNumber.BitLen() > 16*8 {
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
			s.opt.Log.Send(err)
		}
	}()

	return
}

func (s *TlsServer) acceptConnections(ctx context.Context) {
	s.opt.Debug.Info("Starting TCP server at %s", s.opt.Address)
	defer s.opt.Debug.Info("Stopping TCP server at %s", s.opt.Address)

	// s.opt.Log.Info("Starting TCP server at %s", s.opt.Address).Send()
	// defer s.opt.Log.Info("Stopping TCP server at %s", s.opt.Address).Send()

	for {
		conn, err := s.listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			s.opt.Log.Send(err)
			continue
		}

		// s.opt.Log.Debug("Incoming TCP connection from %s", addrIp(conn.RemoteAddr())).Send()
		s.opt.Debug.Debug("Incoming TCP connection from %s", addrIp(conn.RemoteAddr()))

		if tlsConn, ok := conn.(*tls.Conn); ok {
			go func(tlsConn *tls.Conn) {
				log := s.opt.Log.Logger().Tag(addrIp(tlsConn.RemoteAddr()))
				defer log.Drop()

				tlsConn.NetConn()

				if err := s.handleConnection(ctx, tlsConn, log); err != nil {
					if err == io.EOF {
						// log.Debug("Connection closed by client").Send()
						s.opt.Debug.Debug("Connection closed")
					} else {
						s.opt.Debug.Error(err)
						// log.Send(err)
					}
				}
			}(tlsConn)
		} else {
			if err := conn.Close(); err != nil {
				s.opt.Debug.Notice("Error for %s: %s", addrIp(conn.RemoteAddr()), err.Error())
				// s.opt.Log.Notice(err.Error()).Tag(addrIp(conn.RemoteAddr())).Send()
			} else {
				s.opt.Debug.Notice("Connection from %s not TLS - closed by server", addrIp(conn.RemoteAddr()))
				// s.opt.Log.Notice("Connection not TLS - closed by server").Tag(addrIp(conn.RemoteAddr())).Send()
			}

		}
	}
}

func (s *TlsServer) handleConnection(ctx context.Context, tlsConn *tls.Conn, log *logger.Logger) (err error) {
	defer tlsConn.Close()

	if err = tlsConn.HandshakeContext(ctx); err != nil {
		return
	}

	conn, err := s.acquireConn(tlsConn, log)

	if err != nil {
		return
	}

	defer s.releaseConn(conn)

	err = conn.listen(ctx)

	if conn.entriesReceived > 0 || conn.pingsReceived > 0 {
		log.Info("Finished connection").
			Metric("entriesReceived", conn.entriesReceived).
			Metric("entriesSucceeded", conn.entriesSucceeded).
			Metric("pingsReceived", conn.pingsReceived).
			Metric("pongsSent", conn.pongsSent).
			Metric("secondsConnected", int32(s.opt.Clock.UnixNow()-conn.timeConnected)).
			Metric("secondsIdle", int32(s.opt.Clock.UnixNow()-conn.timeLastActive)).
			Send()
	}

	return
}

func (s *TlsServer) acquireConn(tlsConn *tls.Conn, log *logger.Logger) (conn *tlsServerConn, err error) {
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

	var certId uuid.UUID
	cert.SerialNumber.FillBytes(certId[:])

	conn.validBucketIds = cert.SubjectKeyId
	conn.conn = tlsConn
	conn.ack = state.NegotiatedProtocol == protoAck
	conn.log = log.Tag(certId)
	conn.debug = s.opt.Debug
	conn.timeConnected = s.opt.Clock.UnixNow()
	conn.timeLastActive = conn.timeConnected

	return
}

func (s *TlsServer) releaseConn(conn *tlsServerConn) {
	conn.conn.Close()
	conn.conn = nil
	conn.validBucketIds = nil
	conn.log = nil
	conn.debug = nil
	conn.pingsReceived = 0
	conn.pongsSent = 0
	conn.entriesReceived = 0
	conn.entriesSucceeded = 0
	s.connPool.Put(conn)
}
