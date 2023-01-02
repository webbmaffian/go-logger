package logger

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/webbmaffian/go-logger/auth"
)

func NewServer(opt ServerOptions) Server {
	return &server{
		opt: opt,
	}
}

type Server interface {
	Listen(ctx context.Context) (err error)
}

type Authenticator interface {
	Authenticate(ctx context.Context, clientId uuid.UUID, certId uuid.UUID) (err error)
}

type RawEntryReader interface {
	Read(tenantId uint32, b []byte) error
}

type ServerOptions struct {
	Host           string
	Port           int
	Authenticator  Authenticator
	RawEntryReader RawEntryReader
	RootCa         auth.Certificate
}

type server struct {
	opt ServerOptions
}

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidClientId     = errors.New("invalid client ID")
)

func (s *server) Listen(ctx context.Context) (err error) {
	var address strings.Builder
	address.Grow(len(s.opt.Host) + 5)
	address.WriteString(s.opt.Host)
	address.WriteByte(':')
	address.WriteString(strconv.Itoa(s.opt.Port))

	rootCa, err := s.opt.RootCa.Parse()

	if err != nil {
		return
	}

	var rootCaPool x509.CertPool
	rootCaPool.AddCert(rootCa)

	listener, err := tls.Listen("tcp", address.String(), &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  &rootCaPool,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.PeerCertificates == nil || cs.PeerCertificates[0] == nil {
				return ErrInvalidCertificate
			}

			cert := cs.PeerCertificates[0]

			if cert.SerialNumber == nil {
				return ErrInvalidSerialNumber
			}

			certId, err := uuid.ParseBytes(cert.SerialNumber.Bytes())

			if err != nil {
				return ErrInvalidSerialNumber
			}

			clientId, err := uuid.ParseBytes(cert.SubjectKeyId)

			if s.opt.Authenticator == nil {
				return nil
			}

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			return s.opt.Authenticator.Authenticate(ctx, clientId, certId)
		},
	})

	if err != nil {
		return
	}

	log.Println("Listening on:", address.String())

	if err != nil {
		log.Println(err)
		return
	}

	for {
		if err = ctx.Err(); err != nil {
			log.Println("Stopped listening:", err)
			break
		}

		var conn net.Conn
		conn, err = listener.Accept()

		log.Println("New client")

		if err != nil {
			log.Println("Stopped listening:", err)
			return
		}

		go s.handleRequest(ctx, conn.(*tls.Conn))
	}

	return listener.Close()
}

func (s *server) handleRequest(ctx context.Context, conn *tls.Conn) {
	var err error
	serverConn := serverConnection{
		authenticator:  s.opt.Authenticator,
		rawEntryReader: s.opt.RawEntryReader,
	}

	if err = serverConn.authenticate(ctx, conn); err != nil {
		log.Println("Closing connection:", err)
		conn.Close()
		return
	}

	for {
		if err = serverConn.readEntries(ctx, conn); err != nil {
			log.Println("Closing connection:", err)
			conn.Close()
			break
		}
	}
}
