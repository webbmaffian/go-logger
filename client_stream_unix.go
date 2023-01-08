package logger

import (
	"context"
	"net"
	"time"
)

type ClientUnix struct {
	Address string
	TimeNow func() time.Time
}

func (opt *ClientUnix) connect(ctx context.Context) (conn net.Conn, err error) {
	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	return net.DialUnix("unix", nil, &net.UnixAddr{
		Name: opt.Address,
		Net:  "unix",
	})
}

func (opt *ClientUnix) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
