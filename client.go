package logger

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

type client struct {
	connector Connector
	dial      func() (net.Conn, error)
	ctx       context.Context
	conn      net.Conn
	clock     func() time.Time
}

type Connector interface {
	dialer(c *client) func() (net.Conn, error)
	clock() func() time.Time
}

func NewClient(ctx context.Context, connector Connector) io.WriteCloser {
	c := &client{
		connector: connector,
		ctx:       ctx,
		clock:     connector.clock(),
	}

	if c.clock == nil {
		c.clock = time.Now
	}

	c.dial = connector.dialer(c)

	return c
}

func (c *client) Write(b []byte) (n int, err error) {
	var timer *time.Timer

loop:
	for {
		if c.conn == nil {
			// c.conn, err = tls.Dial("tcp", c.Address(), c.dialer.Config)
			log.Println("client: connecting")
			c.conn, err = c.dial()

			if err != nil {
				log.Println("client: failed to connect:", err)
			} else {
				log.Println("client: connected")
			}
		}

		if c.conn != nil {
			c.conn.SetWriteDeadline(c.clock().Add(time.Second * 5))
			n, err = c.conn.Write(b)

			if err == nil {
				break
			}

			log.Println("client: failed to write:", err)

			c.Close()
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
	err = c.conn.Close()
	c.conn = nil
	return
}
