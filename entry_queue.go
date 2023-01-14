package logger

import (
	"sync"
	"time"

	"github.com/rs/xid"
)

func newEntryQueue(bufferSize int) *entryQueue {
	return &entryQueue{
		ch: make(chan *Entry, bufferSize),
		pool: sync.Pool{
			New: func() any {
				return new(Entry)
			},
		},
	}
}

type entryQueue struct {
	ch   chan *Entry
	pool sync.Pool
}

func (q *entryQueue) acquireEntry(t time.Time) *Entry {
	e := q.pool.Get().(*Entry)
	e.Id = xid.NewWithTime(t)
	e.Level = 3
	e.TagsCount = 0
	e.MetaCount = 0
	e.StackTraceCount = 0

	return e
}

func (q *entryQueue) releaseEntry(e *Entry) {
	q.pool.Put(e)
}

func (q *entryQueue) nextEntry() (e *Entry, ok bool) {
	e, ok = <-q.ch
	return
}

func (q *entryQueue) putEntry(e *Entry) {
	q.ch <- e
}

func (q *entryQueue) close() {
	close(q.ch)
}
