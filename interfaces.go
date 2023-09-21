package logger

import (
	"context"
	"time"
)

type EntryProcessor interface {
	ProcessEntry(ctx context.Context, entry *Entry) (err error)
}

type Client interface {
	EntryProcessor
	Now() time.Time
}
