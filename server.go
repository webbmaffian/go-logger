package logger

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/kpango/fastime"
)

func NewServer(ctx context.Context, entryReader io.Reader) Server {
	fastTime := fastime.New().StartTimerD(ctx, time.Second)

	return &server{
		ctx:         ctx,
		entryReader: entryReader,
		time:        fastTime,
	}
}

type Server interface {
	Listen(opt Listener) (err error)
}

type Listener interface {
	listen(s *server) (err error)
}

type ServerOptions struct {
	EntryReader io.Reader
}

type server struct {
	ctx          context.Context
	entryReader  io.Reader
	time         fastime.Fastime
	listenConfig net.ListenConfig
}

func (s *server) Listen(listener Listener) (err error) {
	return listener.listen(s)
}

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidSubjectKeyId = errors.New("invalid subject key ID")
)

func EntryReaderCallback(cb func(bucketId uint64, b []byte) error) io.Reader {
	return entryReaderCallback{
		cb: cb,
	}
}

type entryReaderCallback struct {
	cb func(bucketId uint64, b []byte) error
}

func (e entryReaderCallback) Read(b []byte) (n int, err error) {
	err = e.cb(0, b)
	n = len(b)
	return
}
