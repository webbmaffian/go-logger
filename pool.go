package logger

import (
	"context"
	"sync"

	"github.com/rs/xid"
)

type Pool struct {
	loggerPool sync.Pool
	client     Client
	opt        PoolOptions
}

type PoolOptions struct {
	EntryPool          *EntryPool
	BucketId           uint32
	DefaultEntryTTL    uint16
	DefaultMetaTTL     uint16
	StackTraceSeverity Severity
}

func (opt *PoolOptions) setDefaults() {

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

	if opt.EntryPool == nil {
		opt.EntryPool = new(EntryPool)
	}
}

func NewPool(client Client, options ...PoolOptions) (*Pool, error) {
	var opt PoolOptions

	if options != nil {
		opt = options[0]
	}

	opt.setDefaults()

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
	return pool.opt.EntryPool.Acquire()
}

func (pool *Pool) ReleaseEntry(e *Entry) {
	pool.opt.EntryPool.Release(e)
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

func (pool *Pool) CloseClient(ctx context.Context) error {
	if cli, ok := pool.client.(ClientCloser); ok {
		return cli.Close(ctx)
	}

	return nil
}
