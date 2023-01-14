package logger

import "math"

func Meta(key string, value string) meta {
	return meta{key, value}
}

type meta struct {
	key   string
	value string
}

func (m meta) writeEntry(e *Entry) {
	if e.MetaCount < MaxMetaCount {
		e.MetaKeys[e.MetaCount] = truncate(m.key, math.MaxUint8)
		e.MetaValues[e.MetaCount] = truncate(m.value, math.MaxUint16)
		e.MetaCount++
		e.Level = max(e.Level, 6)
	}
}
