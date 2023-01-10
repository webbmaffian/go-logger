package logger

import (
	"encoding/binary"
	"log"
	"net"
)

func (s *server) handleRequest(bucketId uint32, conn net.Conn) (err error) {
	log.Println("incoming connection")
	defer conn.Close()

	var buf [entrySize]byte

	for {
		if _, err = readFull(s.ctx, conn, buf[:2]); err != nil {
			break
		}

		size := binary.BigEndian.Uint16(buf[:2])

		// log.Printf("server: waiting for message of %d bytes\n", size)

		if _, err = readFull(s.ctx, conn, buf[2:size]); err != nil {
			continue
		}

		if err = validateEntryBytes(buf[:size]); err != nil {
			log.Println("server: INVALID MESSAGE")
			break
		}

		if _, err = s.entryReader.Read(buf[:size]); err != nil {
			break
		}
	}

	log.Println("server: closing tcp connection")

	return
}
