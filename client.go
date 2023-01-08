package logger

import (
	"context"
	"io"
	"time"
)

type client struct {
	ctx       context.Context
	connector Connector
}

type Connector interface {
	write(ctx context.Context, b []byte) error
	close() error
}

func NewClient(ctx context.Context, connector Connector) io.WriteCloser {
	return client{
		connector: connector,
		ctx:       ctx,
	}
}

func (c client) Write(b []byte) (n int, err error) {
	var timer *time.Timer

loop:
	for {
		if err = c.connector.write(c.ctx, b); err == nil {
			break
		}

		if timer == nil {
			timer = time.NewTimer(time.Second * 5)
		} else {
			timer.Reset(time.Second * 5)
		}

		select {
		case <-c.ctx.Done():
			break loop
		case <-timer.C:
			continue
		}
	}

	if timer != nil {
		timer.Stop()
	}

	return
}

func (c client) Close() (err error) {
	return c.connector.close()
}
