package logger

type MetaTTL uint16

func (m MetaTTL) writeEntry(e *Entry) {
	e.TtlEntry = uint16(m)
}
