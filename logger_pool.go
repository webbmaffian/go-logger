package logger

import (
	"context"
	"sync"

	"github.com/kpango/fastime"
	"github.com/rs/xid"
)

type LoggerPool struct {
	pool               sync.Pool
	EntryPool          EntryPool
	EntryProcessor     EntryProcessor
	Clock              fastime.Fastime
	BucketId           uint32
	DefaultEntryTTL    uint16
	DefaultMetaTTL     uint16
	StackTraceSeverity Severity
}

func (pool *LoggerPool) Logger() *Logger {
	if l := pool.pool.Get(); l != nil {
		return l.(*Logger)
	}

	return &Logger{
		pool:     pool,
		ttlEntry: pool.DefaultEntryTTL,
		ttlMeta:  pool.DefaultMetaTTL,
	}
}

func (pool *LoggerPool) Release(l *Logger) {
	l.pool = pool
	l.Reset()
	pool.pool.Put(l)
}

func (pool *LoggerPool) Send(err error) (id xid.ID) {
	var (
		e  *Entry
		ok bool
	)

	if e, ok = err.(*Entry); ok {
		return e.Send()
	}

	e = pool.EntryPool.Acquire()
	e.severity = ERR
	parseErrorString(e, err.Error())
	id = e.id
	pool.EntryProcessor.ProcessEntry(context.Background(), e)
	return
}
