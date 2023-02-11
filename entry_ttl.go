package logger

type TTL uint16

func (t TTL) writeEntry(e *Entry) {
	e.TtlEntry = uint16(t)
}
