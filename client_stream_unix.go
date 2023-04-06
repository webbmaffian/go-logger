package logger

import (
	"context"
	"net"
	"time"

	"github.com/kpango/fastime"
)

type ClientUnix struct {
	Address string
	Clock   fastime.Fastime
}

func (opt *ClientUnix) connect(ctx context.Context) (conn net.Conn, err error) {
	return net.DialUnix("unix", nil, &net.UnixAddr{
		Name: opt.Address,
		Net:  "unix",
	})
}

func (opt *ClientUnix) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.Clock.Now().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
