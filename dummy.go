package logger

import (
	"context"
)

var _ EntryProcessor = (*dummyWriter)(nil)

type dummyWriter struct {
	pool EntryPool
}

func (w *dummyWriter) ProcessEntry(_ context.Context, e *Entry) (err error) {
	w.pool.Release(e)
	return
}
