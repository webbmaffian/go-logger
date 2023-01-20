package logger

import "math"

// Key-value pairs of arbitrary meta
func Meta(v ...any) meta {
	return meta(v)
}

type meta []any

func (m meta) writeEntry(e *Entry) {
	e.Level = max(e.Level, 6)

	// Round down to an even number
	l := len(m) - (len(m) % 2)

	for i := 0; i < l; i++ {
		if e.MetaCount < MaxMetaCount {
			break
		}

		if i%2 == 0 {
			e.MetaKeys[e.MetaCount] = truncate(stringify(m[i-1]), math.MaxUint8)
		} else {
			e.MetaValues[e.MetaCount] = truncate(stringify(m[i]), math.MaxUint16)
			e.MetaCount++
		}
	}
}
