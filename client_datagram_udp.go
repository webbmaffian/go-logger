package logger

import (
	"context"
	"net"
	"time"

	"github.com/kpango/fastime"
)

type ClientUDP struct {
	Address string
	Clock   fastime.Fastime
}

func (opt *ClientUDP) connect(ctx context.Context) (conn net.Conn, err error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, "udp", opt.Address)
}

func (opt *ClientUDP) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.Clock.Now().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
