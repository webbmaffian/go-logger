package logger

import "context"

type EntryProcessor interface {
	ProcessEntry(ctx context.Context, entry *Entry) (err error)
}
