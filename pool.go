package logger

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/xid"
)

type Pool struct {
	loggerPool sync.Pool
	entryPool  sync.Pool
	client     Client
	opt        PoolOptions
}

type PoolOptions struct {
	BucketId           uint32
	DefaultEntryTTL    uint16
	DefaultMetaTTL     uint16
	StackTraceSeverity Severity
}

func NewPool(client Client, options ...PoolOptions) (*Pool, error) {
	var opt PoolOptions

	if options != nil {
		opt = options[0]
	}

	if opt.BucketId == 0 {
		opt.BucketId = client.BucketId()

		if opt.BucketId == 0 {
			return nil, errors.New("bucket ID could not be determined by certificate - please provide it in PoolOptions")
		}
	}

	if opt.DefaultEntryTTL == 0 {
		opt.DefaultEntryTTL = 30
	}

	if opt.DefaultMetaTTL == 0 {
		opt.DefaultMetaTTL = opt.DefaultEntryTTL
	}

	// Zero is actually a valid severity, but is frankly stupid
	if opt.StackTraceSeverity == 0 {
		opt.StackTraceSeverity = NOTICE
	}

	return &Pool{
		client: client,
		opt:    opt,
	}, nil
}

func (pool *Pool) Logger() *Logger {
	if l := pool.loggerPool.Get(); l != nil {
		return l.(*Logger)
	}

	return &Logger{
		pool:     pool,
		ttlEntry: pool.opt.DefaultEntryTTL,
		ttlMeta:  pool.opt.DefaultMetaTTL,
	}
}

func (pool *Pool) ReleaseLogger(l *Logger) {
	l.pool = pool
	l.Reset()
	pool.loggerPool.Put(l)
}

func (pool *Pool) Entry() (e *Entry) {
	if v := pool.entryPool.Get(); v != nil {
		e = v.(*Entry)
	} else {
		e = &Entry{
			level: _3_Message,
		}
	}
	return
}

func (pool *Pool) ReleaseEntry(e *Entry) {
	e.Reset()
	pool.entryPool.Put(e)
}

func (pool *Pool) Send(err error) (id xid.ID) {
	var (
		e  *Entry
		ok bool
	)

	if e, ok = err.(*Entry); ok {
		return e.Send()
	}

	e = pool.Entry()
	e.severity = ERR
	parseErrorString(e, err.Error())
	id = e.id
	pool.client.ProcessEntry(context.Background(), e)
	return
}
