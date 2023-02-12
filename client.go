package logger

import (
	"context"
	"log"
	"net"
	"time"
)

type client struct {
	ctx       context.Context
	connector Connector
	conn      net.Conn
	ch        chan *Entry
}

type Connector interface {
	connect(ctx context.Context) (net.Conn, error)
	write(ctx context.Context, conn net.Conn, b []byte) error
}

type ClientOptions struct {
	BufferSize int
}

func NewClient(ctx context.Context, connector Connector, entryPool EntryPool, options ...ClientOptions) EntryProcessor {
	var opt ClientOptions

	if options != nil {
		opt = options[0]
	} else {
		opt.BufferSize = 100
	}

	c := &client{
		connector: connector,
		ctx:       ctx,
		ch:        make(chan *Entry, opt.BufferSize),
	}

	go func() {
		var buf [MaxEntrySize]byte

	loop:
		for {
			select {
			case err := <-ctx.Done():
				log.Println(err)
				break loop
			case e, ok := <-c.ch:
				if ok {
					s := e.Encode(buf[:])

					if err := c.write(buf[:s]); err != nil {
						log.Println(err)
					}
				}

				entryPool.Release(e)
			}
		}

		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
	}()

	return c
}

func (c *client) ProcessEntry(e *Entry) (err error) {
	c.ch <- e
	return
}

func (c *client) write(b []byte) (err error) {
	var timer *time.Timer

loop:
	for {
		if c.conn == nil {
			c.conn, err = c.connector.connect(c.ctx)

			if err != nil {
				log.Println("server: connection error:", err)
			}
		}

		if c.conn != nil {
			if err = c.connector.write(c.ctx, c.conn, b); err == nil {
				break
			}
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
