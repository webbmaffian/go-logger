package logger

import (
	"encoding/binary"
	"net"
	"time"
)

func (s *server) handleDatagram(conn net.PacketConn) (err error) {
	defer conn.Close()
	// log.Println("server: incoming connection")

	entryCtx := s.entryProc.AcquireCtx()
	defer s.entryProc.ReleaseCtx(entryCtx)

	entry := s.entryPool.Acquire()
	defer s.entryPool.Release(entry)

	var buf [MaxEntrySize]byte
	var n int

	for {
		if err = s.ctx.Err(); err != nil {
			break
		}

		// log.Println("server: waiting for message")

		conn.SetReadDeadline(s.opt.TimeNow().Add(time.Second))
		n, _, err = conn.ReadFrom(buf[:])

		if err != nil {
			continue
		}

		if n < 2 {
			continue
		}

		msgLen := int(binary.BigEndian.Uint16(buf[:2]))

		if msgLen != n {
			continue
		}

		if err = entry.Decode(buf[:n], s.opt.NoCopy); err != nil {
			break
		}

		if err = s.entryProc.ProcessEntry(entry, entryCtx); err != nil {
			break
		}
	}

	return
}
