package debug

import (
	"context"
	"io"
	"net"
)

func Read(ctx context.Context, serverAddr string, dst io.Writer) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Resolve the UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)

	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		return
	}

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	var buf [1024]byte

	for {
		n, err := conn.Read(buf[:])

		if err != nil {
			if err == io.EOF {
				err = nil
			}

			return err
		}

		dst.Write(buf[:n])
	}
}
