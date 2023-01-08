package logger

import (
	"log"
	"net"
)

type ServerUnix struct {
	Address string
}

func (opt ServerUnix) listen(s *server) (err error) {
	addr, err := net.ResolveUnixAddr("unix", opt.Address)

	if err != nil {
		return
	}

	listener, err := net.ListenUnix("unix", addr)

	if err != nil {
		return
	}

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	for {
		if err = s.ctx.Err(); err != nil {
			log.Println("server: stopped listening:", err)
			break
		}

		var conn net.Conn
		conn, err = listener.Accept()

		if err != nil {
			log.Println("server:", err)
			continue
		}

		go func() {
			if err := s.handleRequest(0, conn); err != nil {
				log.Println("server:", err)
			}
		}()
	}

	return listener.Close()
}
