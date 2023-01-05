package remote

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/webbmaffian/go-logger/transports/remote/auth"
)

type ServerTLSOptions struct {
	Host          string
	Port          int
	PrivateKey    auth.PrivateKey
	Certificate   auth.Certificate
	RootCa        auth.Certificate
	Authenticator Authenticator
}

type Authenticator interface {
	Authenticate(ctx context.Context, x509Cert *x509.Certificate) (err error)
}

func (s *server) ListenTLS(opt ServerTLSOptions) (err error) {
	if opt.Host == "" {
		opt.Host = "localhost"
	}

	if opt.Port == 0 {
		opt.Port = 4610
	}

	var address strings.Builder
	address.Grow(len(opt.Host) + 5)
	address.WriteString(opt.Host)
	address.WriteByte(':')
	address.WriteString(strconv.Itoa(opt.Port))

	listener, err := tls.Listen("tcp", address.String(), &tls.Config{
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

	if err != nil {
		return
	}

	go func() {
		<-s.ctx.Done()
		log.Println("server: closing TCP...")
		listener.Close()
	}()

	log.Println("server: listening on:", address.String())

	if err != nil {
		log.Println(err)
		return
	}

	for {
		if err = s.ctx.Err(); err != nil {
			log.Println("server: stopped listening:", err)
			break
		}

		var conn net.Conn
		conn, err = listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			if err = s.handleTLSRequest(conn.(*tls.Conn)); err != nil {
				log.Println("server:", err)
			}
		}()
	}

	return listener.Close()
}

func (s *server) handleTLSRequest(conn *tls.Conn) (err error) {
	defer conn.Close()
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

	// We know that the byte length is 8 from `VerifyConnection`
	bucketId := binary.BigEndian.Uint64(cert.SubjectKeyId)

	var sizeBuf [2]byte
	var buf [math.MaxUint16]byte
	var n int

	for {
		log.Println("server: waiting for message size")
		// Close connection if it's been silent for 10 minutes
		if err = conn.SetReadDeadline(s.time.Now().Add(time.Minute * 10)); err != nil {
			return
		}

		if _, err = readFull(s.ctx, conn, sizeBuf[:]); err != nil {
			return err
		}

		log.Printf("server: received: %08b\n", sizeBuf[:])
		log.Printf("server: waiting for message of %d bytes\n", binary.BigEndian.Uint16(sizeBuf[:]))

		// After recieved size of message, wait up to 1 minute for the rest of the message
		if err = conn.SetReadDeadline(s.time.Now().Add(time.Minute)); err != nil {
			return
		}

		if n, err = readFull(s.ctx, conn, buf[:binary.BigEndian.Uint16(sizeBuf[:])]); err != nil {
			return err
		}

		if err = s.opt.EntryReader.Read(bucketId, buf[:n]); err != nil {
			return
		}
	}

	return

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
		if s.ctx.Err() != nil {
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

	return

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
