package logger

type ServerUDP struct {
	Address string
}

func (opt ServerUDP) listen(s *server) (err error) {
	conn, err := s.listenConfig.ListenPacket(s.ctx, "udp", opt.Address)

	if err != nil {
		return
	}

	return s.handleDatagram(conn)
}
