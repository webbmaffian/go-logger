package logger

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	ErrTooShort        = errors.New("entry too short")
	ErrTooLong         = errors.New("entry too long")
	ErrInvalidSeverity = errors.New("invalid severity")
	ErrTooManyTags     = errors.New("too many tags")
	ErrTooManyMeta     = errors.New("too many meta key/value pairs")
	ErrCorruptEntry    = errors.New("corrupt entry")
)

func validateEntryBytes(b []byte) (err error) {
	totalLen := len(b)

	// Must be at least 12 bytes XID + 1 byte severity
	if totalLen < 13 {
		return ErrTooShort
	}

	if totalLen > math.MaxUint16 {
		return ErrTooLong
	}

	// 0. Entry ID
	s := 12

	// 1. Severity
	if b[s] > byte(DEBUG) {
		return ErrInvalidSeverity
	}
	s++

	l := 1

loop:
	for s < totalLen {
		l++

		switch l {

		case 2: // Message
			s += int(b[s]) + 1

		case 3: // Category
			s += int(b[s]) + 1

		case 4: // Proc ID
			s += int(b[s]) + 1

		case 5: // Tags
			tagsCount := int(b[s])
			if tagsCount > 32 {
				return ErrTooManyTags
			}
			s++

			for i := 0; i < tagsCount; i++ {
				s += int(b[s]) + 1
			}

		case 6: // Meta
			metaCount := int(b[s])
			if metaCount > 32 {
				return ErrTooManyMeta
			}
			s++

			for i := 0; i < metaCount; i++ {
				if s >= totalLen {
					return ErrCorruptEntry
				}

				s += int(b[s]) + 1
				s += int(binary.BigEndian.Uint16(b[s:s+1])) + 2
			}

		default:
			break loop
		}
	}

	if s != totalLen {
		err = ErrCorruptEntry
	}

	return
}
