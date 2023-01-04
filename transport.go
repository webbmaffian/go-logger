package logger

import (
	"io"
	"time"
)

type Transport interface {
	io.WriteCloser
	SetNowFunc(func() time.Time)
}
