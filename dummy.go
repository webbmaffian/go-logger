package logger

import (
	"context"
	"io"
	"sync"
)

type dummyWriter struct {
	buf       [entrySize]byte
	entryPool sync.Pool
}

func (d *dummyWriter) AcquireEntry() *Entry {
	return d.entryPool.Get().(*Entry)
}

func (d *dummyWriter) Write(e *Entry) {
	e.Encode(d.buf[:])
	d.entryPool.Put(e)
}

func (d *dummyWriter) Close() error {
	return nil
}

func (d *dummyWriter) ReadFrom(r io.Reader) (n int64, err error) {
	s, err := r.Read(d.buf[:])
	n = int64(s)
	return
}

type dummyAuthenticator struct{}

func (d dummyAuthenticator) LoadClientSecret(ctx context.Context, clientId []byte, clientSecret []byte) error {
	return nil
}

type dummyRawEntryReader struct{}

func (d dummyRawEntryReader) Read(clientId []byte, b []byte) error {
	return nil
}
