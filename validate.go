package logger

import (
	"encoding/binary"
	"errors"
)

var (
	ErrTooShort          = errors.New("entry too short")
	ErrTooLong           = errors.New("entry too long")
	ErrInvalidSeverity   = errors.New("invalid severity")
	ErrTooManyTags       = errors.New("too many tags")
	ErrTooManyMetric     = errors.New("too many metric key/value pairs")
	ErrTooManyMeta       = errors.New("too many meta key/value pairs")
	ErrTooManyStackTrace = errors.New("too many stack traces")
	ErrCorruptEntry      = errors.New("corrupt entry")
)

func validateEntryBytes(b []byte) (err error) {
	if len(b) < 2 {
		return ErrTooShort
	}

	if len(b) > MaxEntrySize {
		return ErrTooLong
	}

	var s uint16
	totalLen := uint16(len(b))
	total := binary.BigEndian.Uint16(b[s:])
	s += 2

	if uint16(totalLen) != total {
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

		case _4_CategoryId:
			s++

		case _5_Tags:
			tagsCount := int(b[s])
			if tagsCount > MaxTagsCount {
				return ErrTooManyTags
			}
			s++

			for i := 0; i < tagsCount; i++ {
				s += uint16(b[s]) + 1
			}

		case _6_Metric:
			metricCount := int(b[s])
			if metricCount > MaxMetricCount {
				return ErrTooManyMetric
			}
			s++

			for i := 0; i < metricCount; i++ {
				if s >= totalLen {
					return ErrCorruptEntry
				}

				s += uint16(b[s]) + 1
				s += 4
			}

		case _7_Meta:
			metaCount := int(b[s])
			if metaCount > MaxMetaCount {
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

		case _8_Stack_trace:
			traceCount := int(b[s])
			if traceCount > MaxStackTraceCount {
				return ErrTooManyStackTrace
			}
			s++

			for i := 0; i < traceCount; i++ {
				if s >= totalLen {
					return ErrCorruptEntry
				}

				s += uint16(b[s]) + 1
				s += 2
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
