package logger

import (
	"context"
	"errors"
	"time"

	"github.com/kpango/fastime"
)

func NewServer(ctx context.Context, opt ServerOptions) Server {
	fastTime := fastime.New().StartTimerD(ctx, time.Second)

	return &server{
		ctx:  ctx,
		opt:  opt,
		time: fastTime,
	}
}

type Server interface {
	ListenTLS(opt ServerTLSOptions) (err error)
	// ListenTCP(opt ServerTCPOptions) (err error)
	// ListenUDP(opt ServerUDPOptions) (err error)
}

type EntryReader interface {
	Read(bucketId uint64, b []byte) error
}

type ServerOptions struct {
	EntryReader EntryReader
}

type server struct {
	ctx  context.Context
	opt  ServerOptions
	time fastime.Fastime
}

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidSubjectKeyId = errors.New("invalid subject key ID")
)

func EntryReaderCallback(cb func(bucketId uint64, b []byte) error) EntryReader {
	return entryReaderCallback{
		cb: cb,
	}
}

type entryReaderCallback struct {
	cb func(bucketId uint64, b []byte) error
}

func (e entryReaderCallback) Read(bucketId uint64, b []byte) error {
	return e.cb(bucketId, b)
}
