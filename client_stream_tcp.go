package logger

import (
	"context"
	"net"
	"time"

	"github.com/kpango/fastime"
)

type ClientTCP struct {
	Address string
	Clock   fastime.Fastime
	dialer  net.Dialer
}

func (opt *ClientTCP) connect(ctx context.Context) (conn net.Conn, err error) {
	if conn, err = opt.dialer.DialContext(ctx, "tcp", opt.Address); err == nil {
		c := conn.(*net.TCPConn)

		c.CloseRead()
	}

	return
}

func (opt *ClientTCP) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.Clock.Now().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
