package logger

import "sync"

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

func (q *entryQueue) acquireEntry() *Entry {
	return q.pool.Get().(*Entry)
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
