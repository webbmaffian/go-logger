package logger

import (
	"unsafe"
)

var _ EntryProcessor = (*dummyWriter)(nil)

type dummyWriter struct {
	pool EntryPool
}

func (w *dummyWriter) ProcessEntry(e *Entry, _ unsafe.Pointer) (err error) {
	w.pool.Release(e)
	return
}

func (dummyWriter) AcquireCtx() unsafe.Pointer {
	return nil
}

func (dummyWriter) ReleaseCtx(_ unsafe.Pointer) {}
