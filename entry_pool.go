package logger

import (
	"sync"
)

type EntryPool struct {
	pool sync.Pool
}

func (pool *EntryPool) Acquire() (e *Entry) {
	if v := pool.pool.Get(); v != nil {
		e = v.(*Entry)
	} else {
		e = &Entry{
			level: _3_Message,
		}
	}
	return
}

func (pool *EntryPool) Release(e *Entry) {
	e.Reset()
	pool.pool.Put(e)
}
