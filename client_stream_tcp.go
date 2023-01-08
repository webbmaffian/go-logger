package logger

import (
	"context"
	"net"
	"time"
)

type ClientTCP struct {
	Address string
	TimeNow func() time.Time
	dialer  net.Dialer
}

func (opt *ClientTCP) connect(ctx context.Context) (conn net.Conn, err error) {
	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	if conn, err = opt.dialer.DialContext(ctx, "tcp", opt.Address); err == nil {
		c := conn.(*net.TCPConn)

		c.CloseRead()
	}

	return
}

func (opt *ClientTCP) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
