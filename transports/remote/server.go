package remote

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/webbmaffian/go-logger/transports/remote/auth"
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
	Authenticate(ctx context.Context, x509Cert *x509.Certificate) (err error)
}

type EntryReader interface {
	Read(bucketId uint64, b []byte) error
}

type ServerOptions struct {
	Host          string
	Port          int
	Authenticator Authenticator
	EntryReader   EntryReader
	Certificate   auth.Certificate
	RootCa        auth.Certificate
	PrivateKey    auth.PrivateKey
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

	listener, err := tls.Listen("tcp", address.String(), &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    s.opt.RootCa.X509Pool(),
		Certificates: s.opt.Certificate.TLSChain(s.opt.PrivateKey),
		NextProtos:   []string{"wallaaa"},
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.PeerCertificates == nil || cs.PeerCertificates[0] == nil {
				return ErrInvalidCertificate
			}

			cert := cs.PeerCertificates[0]

			if cert.SerialNumber == nil {
				return ErrInvalidSerialNumber
			}

			if s.opt.Authenticator == nil {
				return nil
			}

			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			return s.opt.Authenticator.Authenticate(ctx, cert)
		},
	})

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		log.Println("server: closing TCP...")
		listener.Close()
	}()

	log.Println("server: listening on:", address.String())

	if err != nil {
		log.Println(err)
		return
	}

	for {
		if err = ctx.Err(); err != nil {
			log.Println("server: stopped listening:", err)
			break
		}

		var conn net.Conn
		conn, err = listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		go s.handleRequest(ctx, conn.(*tls.Conn))
	}

	return listener.Close()
}

func (s *server) handleRequest(ctx context.Context, conn *tls.Conn) {
	defer conn.Close()
	log.Println("server: incoming connection")

	if err := conn.HandshakeContext(ctx); err != nil {
		log.Println("server: error:", err)
		return
	} else {
		log.Println("server: handshake successful")
	}

	state := conn.ConnectionState()
	log.Println("server: resumed:", state.DidResume)
	log.Println("server: TLS version:", tlsVersion(state.Version))
	log.Println("server: Server name:", state.ServerName)
	log.Println("server: Handshake complete:", state.HandshakeComplete)
	log.Println("server: Negotiated protocol:", state.NegotiatedProtocol)

	for i, cert := range state.PeerCertificates {
		log.Println("server: received certificate", i, ":\n", auth.CertificateX509(cert))
	}

	var b [10]byte

	for {
		if ctx.Err() != nil {
			break
		}

		n, err := conn.Read(b[:])

		if err != nil {
			log.Println("server: error:", err)
			break
		} else {
			log.Printf("server: %s\n", b[:n])
		}

	}

	// var err error
	// serverConn := serverConnection{
	// 	authenticator:  s.opt.Authenticator,
	// 	rawEntryReader: s.opt.RawEntryReader,
	// }

	// if err = serverConn.authenticate(ctx, conn); err != nil {
	// 	log.Println("Closing connection:", err)
	// 	conn.Close()
	// 	return
	// }

	// for {
	// 	if err = serverConn.readEntries(ctx, conn); err != nil {
	// 		log.Println("Closing connection:", err)
	// 		conn.Close()
	// 		break
	// 	}
	// }
}

func tlsVersion(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	default:
		return fmt.Sprintf("unknown (%d)", v)
	}
}
