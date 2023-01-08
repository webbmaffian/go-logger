package logger

import (
	"context"
	"net"
	"time"
)

type ClientUDP struct {
	Address string
	TimeNow func() time.Time
}

func (opt *ClientUDP) connect(ctx context.Context) (conn net.Conn, err error) {
	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	var dialer net.Dialer
	return dialer.DialContext(ctx, "udp", opt.Address)
}

func (opt *ClientUDP) write(ctx context.Context, conn net.Conn, b []byte) (err error) {
	conn.SetWriteDeadline(opt.TimeNow().Add(time.Second * 5))
	_, err = conn.Write(b)
	return
}
