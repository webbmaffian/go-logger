package logger

import "unsafe"

type EntryProcessor interface {
	ProcessEntry(entry *Entry, entryCtx unsafe.Pointer) (err error)
	AcquireCtx() unsafe.Pointer
	ReleaseCtx(p unsafe.Pointer)
}
