package logger

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/webbmaffian/go-logger/auth"
)

type ServerTLS struct {
	Address       string
	PrivateKey    auth.PrivateKey
	Certificate   auth.Certificate
	RootCa        auth.Certificate
	Authenticator Authenticator
}

type Authenticator interface {
	Authenticate(ctx context.Context, x509Cert *x509.Certificate) (err error)
}

func (opt ServerTLS) listen(s *server) (err error) {
	netListener, err := s.listenConfig.Listen(s.ctx, "tcp", opt.Address)

	if err != nil {
		return
	}

	listener := tls.NewListener(netListener, &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    opt.RootCa.X509Pool(),
		Certificates: opt.Certificate.TLSChain(opt.PrivateKey),
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.PeerCertificates == nil || cs.PeerCertificates[0] == nil {
				return ErrInvalidCertificate
			}

			cert := cs.PeerCertificates[0]

			if cert.SerialNumber == nil {
				return ErrInvalidSerialNumber
			}

			// Ensure `SubjectKeyID` contains one or more uint32
			if cert.SubjectKeyId == nil || len(cert.SubjectKeyId)%4 != 0 {
				return ErrInvalidSubjectKeyId
			}

			if opt.Authenticator == nil {
				return nil
			}

			ctx, cancel := context.WithTimeout(s.ctx, time.Second)
			defer cancel()

			return opt.Authenticator.Authenticate(ctx, cert)
		},
	})

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	for {
		if err = s.ctx.Err(); err != nil {
			log.Println("server: stopped listening:", err)
			break
		}

		var conn net.Conn
		conn, err = listener.Accept()

		if err != nil {
			log.Println("server:", err)
			continue
		}

		go func() {
			if err := s.handleTLSRequest(conn.(*tls.Conn)); err != nil && err != io.EOF {
				// if s.opt.Logger.core != nil {
				// 	s.opt.Logger.Notice(err.Error())
				// }
			}
		}()
	}

	return listener.Close()
}

func (s *server) handleTLSRequest(conn *tls.Conn) (err error) {
	if err = conn.HandshakeContext(s.ctx); err != nil {
		// if s.opt.Logger.core != nil {
		// 	s.opt.Logger.Debug("failed TLS handshake", addrToIp(conn.RemoteAddr()).String(), Meta("_", err.Error()))
		// }

		return
	}

	// Handshake done - we won't write any more data to TCP
	if err = conn.CloseWrite(); err != nil {
		return
	}

	state := conn.ConnectionState()

	if state.PeerCertificates == nil {
		return errors.New("server: missing peer certificates")
	}

	cert := state.PeerCertificates[0]

	// if s.opt.Logger.core != nil {
	// 	s.opt.Logger.Debug("successful TLS handshake with certificate %s", bigIntToUuid(cert.SerialNumber), addrToIp(conn.RemoteAddr()).String())
	// }

	return s.handleRequest(conn, cert.SubjectKeyId)
}
