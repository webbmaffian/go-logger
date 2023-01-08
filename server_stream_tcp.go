package logger

import (
	"log"
	"net"
)

type ServerTCP struct {
	Address string
}

func (opt ServerTCP) listen(s *server) (err error) {
	netListener, err := s.listenConfig.Listen(s.ctx, "tcp", opt.Address)

	if err != nil {
		return
	}

	listener := netListener.(*net.TCPListener)

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	for {
		var conn *net.TCPConn

		if conn, err = listener.AcceptTCP(); err != nil {
			break
		}

		go func() {
			if err := s.handleTCPRequest(conn); err != nil {
				log.Println("server:", err)
			}
		}()
	}

	if err = listener.Close(); err == net.ErrClosed {
		err = nil
	}

	return
}

func (s *server) handleTCPRequest(conn *net.TCPConn) (err error) {
	log.Println("server: incoming connection")

	// We will never write to this connection
	if err = conn.CloseWrite(); err != nil {
		log.Println("server:", err)
		conn.Close()
		return
	}

	// We don't know any bucket ID
	return s.handleRequest(0, conn)
}
