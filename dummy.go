package logger

import (
	"context"
	"time"

	"github.com/kpango/fastime"
)

var _ Client = (*dummyWriter)(nil)

type dummyWriter struct {
	clock fastime.Fastime
}

func (w *dummyWriter) ProcessEntry(_ context.Context, _ *Entry) (err error) {
	return
}

func (w *dummyWriter) Now() time.Time {
	return w.clock.Now()
}

func (w *dummyWriter) BucketId() uint32 {
	return 1
}
