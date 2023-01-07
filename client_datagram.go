package logger

import (
	"net"
	"time"
)

type ClientUDP struct {
	Address string
	Clock   func() time.Time
}

func (opt ClientUDP) dialer(c *client) func() (net.Conn, error) {
	var dialer net.Dialer

	return func() (net.Conn, error) {
		return dialer.DialContext(c.ctx, "udp", opt.Address)
	}
}

func (opt ClientUDP) clock() func() time.Time {
	return opt.Clock
}

type ClientUnixgram struct {
	Address string
	Clock   func() time.Time
}

func (opt ClientUnixgram) dialer(c *client) func() (net.Conn, error) {
	var dialer net.Dialer

	return func() (net.Conn, error) {
		return dialer.DialContext(c.ctx, "unixgram", opt.Address)
	}
}

func (opt ClientUnixgram) clock() func() time.Time {
	return opt.Clock
}
