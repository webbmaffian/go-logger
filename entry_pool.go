package logger

import (
	"sync"
)

func NewEntryPool() EntryPool {
	return &entryPool{
		pool: sync.Pool{
			New: func() any {
				return &Entry{
					Level: _3_Message,
				}
			},
		},
	}
}

type EntryPool interface {
	Acquire() *Entry
	Release(e *Entry)
}

type entryPool struct {
	pool sync.Pool
}

func (q *entryPool) Acquire() *Entry {
	return q.pool.Get().(*Entry)
}

func (q *entryPool) Release(e *Entry) {
	e.Reset()
	q.pool.Put(e)
}
