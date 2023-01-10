package logger

import (
	"encoding/binary"
	"net"
	"time"
)

func (s *server) handleDatagram(conn net.PacketConn) (err error) {
	defer conn.Close()
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

		// if err = s.opt.EntryReader.ReadEntry(0, buf[2:n]); err != nil {
		// 	return
		// }
	}
}
