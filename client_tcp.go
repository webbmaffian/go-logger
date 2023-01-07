package logger

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type ClientTCPOptions struct {
	Host string
	Port int
}

type tcpClient struct {
	opt    ClientTCPOptions
	ctx    context.Context
	conn   net.Conn
	time   func() time.Time
	dialer net.Dialer
}

func NewTCPClient(ctx context.Context, opt ClientTCPOptions) Transport {
	return &tcpClient{
		ctx:  ctx,
		opt:  opt,
		time: time.Now,
	}
}

func (c *tcpClient) SetNowFunc(f func() time.Time) {
	c.time = f
}

func (c tcpClient) Address() string {
	var b strings.Builder
	b.Grow(len(c.opt.Host) + 5)
	b.WriteString(c.opt.Host)
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(c.opt.Port))
	return b.String()
}

func (c *tcpClient) Write(p []byte) (n int, err error) {
	var timer *time.Timer

loop:
	for {
		if c.conn == nil {
			// c.conn, err = tls.Dial("tcp", c.Address(), c.dialer.Config)
			log.Println("client: connecting")
			c.conn, err = c.dialer.DialContext(c.ctx, "tcp", c.Address())

			if err != nil {
				log.Println("client: failed to connect:", err)
			} else {
				log.Println("client: connected")
			}
		}

		if c.conn != nil {
			c.conn.SetWriteDeadline(c.time().Add(time.Second * 5))
			n, err = c.conn.Write(p)

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

func (c *tcpClient) Close() (err error) {
	err = c.conn.Close()
	c.conn = nil
	return
}
