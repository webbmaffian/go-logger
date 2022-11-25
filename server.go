package logger

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"
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
	LoadClientSecret(ctx context.Context, clientId []byte, clientSecret []byte) error
}

type RawEntryReader interface {
	Read(b []byte) error
}

type ServerOptions struct {
	Host           string
	Port           int
	Authenticator  Authenticator
	RawEntryReader RawEntryReader
}

type server struct {
	listener net.Listener
	opt      ServerOptions
}

func (s *server) Listen(ctx context.Context) (err error) {
	var address strings.Builder
	address.Grow(len(s.opt.Host) + 5)
	address.WriteString(s.opt.Host)
	address.WriteByte(':')
	address.WriteString(strconv.Itoa(s.opt.Port))

	s.listener, err = net.Listen("tcp", address.String())

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
		conn, err = s.listener.Accept()

		log.Println("New client")

		if err != nil {
			log.Println("Stopped listening:", err)
			return
		}

		go s.handleRequest(ctx, conn)
	}

	return s.listener.Close()
}

func (s *server) handleRequest(ctx context.Context, conn net.Conn) {
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
