package logger

import (
	"io"
)

var _ io.WriteCloser = dummyWriter{}

type dummyWriter struct{}

func (d dummyWriter) Write(b []byte) (n int, err error) {
	return
}

func (d dummyWriter) Close() error {
	return nil
}
