package logger

import (
	"encoding/binary"
	"errors"
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
	if len(b) < 2 {
		return ErrTooShort
	}

	if len(b) > entrySize {
		return ErrTooLong
	}

	var s uint16
	totalLen := uint16(len(b))
	total := binary.BigEndian.Uint16(b[s:])
	s += 2

	if uint16(totalLen) != total+2 {
		return ErrCorruptEntry
	}

	var l level

	for s < totalLen {
		switch l {

		case _0_BucketId:
			s += 4

		case _1_EntryId:
			s += 12

		case _2_Severity:
			if b[s] > byte(DEBUG) {
				return ErrInvalidSeverity
			}
			s++

		case _3_Message:
			s += uint16(b[s]) + 1

		case _4_Category:
			s += uint16(b[s]) + 1

		case _5_ProcId:
			s += uint16(b[s]) + 1

		case _6_Tags:
			tagsCount := int(b[s])
			if tagsCount > 32 {
				return ErrTooManyTags
			}
			s++

			for i := 0; i < tagsCount; i++ {
				s += uint16(b[s]) + 1
			}

		case _7_Meta:
			metaCount := int(b[s])
			if metaCount > 32 {
				return ErrTooManyMeta
			}
			s++

			for i := 0; i < metaCount; i++ {
				if s >= totalLen {
					return ErrCorruptEntry
				}

				s += uint16(b[s]) + 1
				s += binary.BigEndian.Uint16(b[s:s+2]) + 2
			}

		default:
			return ErrCorruptEntry
		}

		l++
	}

	if s != totalLen {
		err = ErrCorruptEntry
	}

	return
}
