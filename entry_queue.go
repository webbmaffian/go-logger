package logger

import "sync"

func newEntryQueue(bufferSize int) *entryQueue {
	return &entryQueue{
		ch: make(chan *entry, bufferSize),
		pool: sync.Pool{
			New: func() any {
				return new(entry)
			},
		},
	}
}

type entryQueue struct {
	ch   chan *entry
	pool sync.Pool
}

func (q *entryQueue) acquireEntry() *entry {
	return q.pool.Get().(*entry)
}

func (q *entryQueue) releaseEntry(e *entry) {
	q.pool.Put(e)
}

func (q *entryQueue) nextEntry() (e *entry, ok bool) {
	e, ok = <-q.ch
	return
}

func (q *entryQueue) putEntry(e *entry) {
	q.ch <- e
}

func (q *entryQueue) close() {
	close(q.ch)
}
