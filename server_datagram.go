package logger

import (
	"encoding/binary"
	"net"
	"time"
)

type ServerUDP struct {
	Address string
}

func (opt ServerUDP) listen(s *server) (err error) {
	conn, err := s.listenConfig.ListenPacket(s.ctx, "udp", opt.Address)

	if err != nil {
		return
	}

	return s.handleDatagram(conn)
}

type ServerUnixgram struct {
	Address string
}

func (opt ServerUnixgram) listen(s *server) (err error) {
	// conn, err := s.listenConfig.ListenPacket(s.ctx, "unixgram", opt.Address)
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{
		Name: opt.Address,
		Net:  "unixgram",
	})

	if err != nil {
		return
	}

	return s.handleDatagram(conn)
}

func (s *server) handleDatagram(conn net.PacketConn) (err error) {
	// log.Println("server: incoming connection")

	var buf [entrySize]byte
	var n int

	for {
		if err = s.ctx.Err(); err != nil {
			return
		}

		// log.Println("server: waiting for message")

		conn.SetReadDeadline(s.time.Now().Add(time.Second))
		n, _, err = conn.ReadFrom(buf[:])

		if err != nil {
			continue
		}

		if n < 2 {
			continue
		}

		msgLen := int(binary.BigEndian.Uint16(buf[:2]))

		if msgLen != n-2 {
			continue
		}

		if err = s.opt.EntryReader.Read(0, buf[2:n]); err != nil {
			return
		}
	}
}
