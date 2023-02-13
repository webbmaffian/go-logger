package logger

// Key-value pairs of arbitrary meta
func Meta(key string, val any) meta {
	return meta{key, val}
}

type meta struct {
	key string
	val any
}

func (m meta) writeEntry(e *Entry) {
	e.Level = max(e.Level, _7_Meta)

	if e.MetaCount < MaxMetaCount {
		e.MetaKeys[e.MetaCount] = truncate(m.key, MaxMetaKeySize)
		e.MetaValues[e.MetaCount] = truncate(stringify(m.val), MaxMetaValueSize)
		e.MetaCount++
	}
}
