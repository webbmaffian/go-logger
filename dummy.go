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

func (d *dummyWriter) AcquireEntry() *entry {
	return d.entryPool.Get().(*entry)
}

func (d *dummyWriter) Write(e *entry) {
	e.encode(d.buf[:])
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

func (d dummyRawEntryReader) Read(b []byte) error {
	return nil
}
