package logger

import (
	"net"
	"os"
)

type ServerUnixgram struct {
	Address string
}

func (opt ServerUnixgram) listen(s *server) (err error) {
	// conn, err := s.listenConfig.ListenPacket(s.ctx, "unixgram", opt.Address)

	addr, err := net.ResolveUnixAddr("unixgram", opt.Address)

	if err != nil {
		return
	}

	conn, err := net.ListenUnixgram("unixgram", addr)

	if err != nil {
		return
	}

	defer os.Remove(opt.Address)

	return s.handleDatagram(conn)
}
