package debug

import (
	"fmt"
	"log"
	"net"
	"sync"
)

func NewWriter(serverAddr string) (sock *Writer) {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)

	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error dialing UDP:", err)
		return
	}

	if err != nil {
		log.Println("2. failed to open debug socket:", err)
		return
	}

	log.Println("opened debug socket at", serverAddr)

	return &Writer{
		conn: conn,
	}
}

type Writer struct {
	conn net.Conn
	mu   sync.Mutex
}

func (sock *Writer) Write(format string, args ...any) {
	sock.mu.Lock()
	defer sock.mu.Unlock()

	if _, err := fmt.Fprintf(sock.conn, format, args...); err != nil {
		log.Println("failed to write to debug socket")
	}
}

func (sock *Writer) Close() (err error) {
	sock.mu.Lock()
	defer sock.mu.Unlock()

	err = sock.conn.Close()
	sock.conn = nil
	return
}
