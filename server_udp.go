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

	var sizeBuf [2]byte
	var buf [entrySize]byte
	var n int

	for {
		log.Println("server: waiting for message size")
		// Close connection if it's been silent for 10 minutes
		if err = conn.SetReadDeadline(s.time.Now().Add(time.Minute * 10)); err != nil {
			return
		}

		if _, err = readFullPackets(s.ctx, conn, sizeBuf[:]); err != nil {
			return err
		}

		log.Printf("server: received: %08b\n", sizeBuf[:])
		log.Printf("server: waiting for message of %d bytes\n", binary.BigEndian.Uint16(sizeBuf[:]))

		// After recieved size of message, wait up to 1 minute for the rest of the message
		if err = conn.SetReadDeadline(s.time.Now().Add(time.Minute)); err != nil {
			return
		}

		if n, err = readFullPackets(s.ctx, conn, buf[:binary.BigEndian.Uint16(sizeBuf[:])]); err != nil {
			return err
		}

		if err = s.opt.EntryReader.Read(0, buf[:n]); err != nil {
			return
		}
	}
}
