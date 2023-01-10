package logger

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
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
		NextProtos:   []string{"wallaaa"},
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.PeerCertificates == nil || cs.PeerCertificates[0] == nil {
				return ErrInvalidCertificate
			}

			cert := cs.PeerCertificates[0]

			if cert.SerialNumber == nil {
				return ErrInvalidSerialNumber
			}

			// Ensure `SubjectKeyID` is a uint64
			if len(cert.SubjectKeyId) != 8 {
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
			if err := s.handleTLSRequest(conn.(*tls.Conn)); err != nil {
				log.Println("server:", err)
			}
		}()
	}

	return listener.Close()
}

func (s *server) handleTLSRequest(conn *tls.Conn) (err error) {
	log.Println("server: incoming connection")

	if err = conn.HandshakeContext(s.ctx); err != nil {
		return
	} else {
		log.Println("server: handshake successful")
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

	return s.handleRequest(binary.BigEndian.Uint32(cert.SubjectKeyId), conn)
}
