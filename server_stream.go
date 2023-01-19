package logger

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func (s *server) handleRequest(bucketId uint32, conn net.Conn) (err error) {
	log.Println("incoming connection")
	defer conn.Close()

	var buf [MaxEntrySize]byte

	for {
		// if err = s.ctx.Err(); err != nil {
		// 	return
		// }

		conn.SetReadDeadline(s.time.Now().Add(time.Second))
		if _, err = io.ReadFull(conn, buf[:2]); err != nil {
			break
		}

		size := binary.BigEndian.Uint16(buf[:2])

		// log.Printf("server: waiting for message of %d bytes\n", size)
		// log.Println("Reading", size, "bytes...")

		conn.SetReadDeadline(s.time.Now().Add(time.Second * 5))
		if _, err = io.ReadFull(conn, buf[2:size]); err != nil {
			continue
		}

		if err = validateEntryBytes(buf[:size]); err != nil {
			log.Println("server: INVALID MESSAGE:", err)
			log.Println(buf[:size])
			break
		}

		if _, err = s.entryReader.Read(buf[:size]); err != nil {
			break
		}
	}

	log.Println("server: closing tcp connection")

	return
}
