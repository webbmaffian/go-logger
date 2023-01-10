package logger

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

type client struct {
	ctx       context.Context
	connector Connector
	conn      net.Conn
}

type Connector interface {
	connect(ctx context.Context) (net.Conn, error)
	write(ctx context.Context, conn net.Conn, b []byte) error
}

func NewClient(ctx context.Context, connector Connector) io.ReadWriteCloser {
	return &client{
		connector: connector,
		ctx:       ctx,
	}
}

func (c *client) Write(b []byte) (n int, err error) {
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

func (c *client) Close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}

	return
}

func (c *client) Read(b []byte) (n int, err error) {
	return c.Write(b)
}
