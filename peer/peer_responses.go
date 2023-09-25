package peer

type respType uint8

const (
	respAckNOK respType = iota
	respAckOK
	respClose
)
