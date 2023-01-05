package logger

import (
	"encoding/binary"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type ServerUDPOptions struct {
	Host string
	Port int
}

func (s *server) ListenUDP(opt ServerUDPOptions) (err error) {
	if opt.Host == "" {
		opt.Host = "localhost"
	}

	if opt.Port == 0 {
		opt.Port = 4610
	}

	var address strings.Builder
	address.Grow(len(opt.Host) + 5)
	address.WriteString(opt.Host)
	address.WriteByte(':')
	address.WriteString(strconv.Itoa(opt.Port))

	listener, err := net.ListenPacket("udp", address.String())

	if err != nil {
		return
	}

	go func() {
		<-s.ctx.Done()
		log.Println("server: closing TCP...")
		listener.Close()
	}()

	log.Println("server: listening on:", address.String())

	if err != nil {
		log.Println(err)
		return
	}

	for {
		if err = s.ctx.Err(); err != nil {
			log.Println("server: stopped listening:", err)
			break
		}

		if err = s.handleUDPRequest(listener); err != nil {
			log.Println("server:", err)
		}
	}

	return listener.Close()
}

func (s *server) handleUDPRequest(conn net.PacketConn) (err error) {
	log.Println("server: incoming connection")

	var buf [entrySize]byte
	var n int

	for {
		if err = s.ctx.Err(); err != nil {
			return
		}

		log.Println("server: waiting for message")

		conn.SetReadDeadline(s.time.Now().Add(time.Second))
		n, _, err = conn.ReadFrom(buf[:])

		if err != nil {
			log.Println(err)
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
