package logger

import (
	"context"
	"net"
	"time"
)

type ClientUnixgram struct {
	Address string
	TimeNow func() time.Time
}

func (opt *ClientUnixgram) connect(ctx context.Context) (conn net.Conn, err error) {
	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	return net.DialUnix("unixgram", nil, &net.UnixAddr{
		Name: opt.Address,
		Net:  "unixgram",
	})
}

func (opt *ClientUnixgram) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
