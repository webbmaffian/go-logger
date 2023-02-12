package logger

import (
	"context"
	"net"
	"time"
)

func NewServer(ctx context.Context, entryProc EntryProcessor, entryPool EntryPool, options ...ServerOptions) Server {
	var opt ServerOptions

	if options != nil {
		opt = options[0]
	}

	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	return &server{
		ctx:       ctx,
		entryProc: entryProc,
		entryPool: entryPool,
		opt:       opt,
	}
}

type Server interface {
	Listen(opt Listener) (err error)
}

type Listener interface {
	listen(s *server) (err error)
}

type ServerOptions struct {
	TimeNow func() time.Time
	NoCopy  bool
}

type server struct {
	ctx          context.Context
	entryProc    EntryProcessor
	entryPool    EntryPool
	listenConfig net.ListenConfig
	opt          ServerOptions
}

func (s *server) Listen(listener Listener) (err error) {
	return listener.listen(s)
}
