package logger

import (
	"context"
	"net"
	"time"

	"github.com/kpango/fastime"
)

type ClientUnixgram struct {
	Address string
	Clock   fastime.Fastime
}

func (opt *ClientUnixgram) connect(ctx context.Context) (conn net.Conn, err error) {
	return net.DialUnix("unixgram", nil, &net.UnixAddr{
		Name: opt.Address,
		Net:  "unixgram",
	})
}

func (opt *ClientUnixgram) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.Clock.Now().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
