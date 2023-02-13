package logger

import (
	"encoding/binary"
	"math"
	"runtime"

	"github.com/rs/xid"
)

/*
	0. BucketId
		4 byte (uint32) integer
	1. EntryId
		12 byte XID
	2. Severity
		1 byte
	3. Message
		1 byte (uint8) length (X)
		X bytes string
	4. CategoryId
		1 byte (uint8)
	5. Tags
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string
	6. Metric
		1 byte (uint8) count (0-32)
			1 byte (uint8) length (X)
			X bytes string key
			2 bytes (int16) value
		[...31]
	7. Meta
		1 byte (uint8) count (0-32)
			1 byte (uint8) length (X)
			X bytes string key
			2 bytes (uint16) length (Y)
			Y bytes string value
		[...31]
	8. Stack trace
		1 byte (uint8) count (0-8)
			1 byte (uint8) path length (X)
			X bytes string path
			2 bytes (uint16) line number
		[...]
	9. TTL: Entry
		2 byte (uint16) days
	10. TTL: Meta
		2 byte (uint16) days

*/

type level uint8

const (
	MaxEntrySize          = 65_507 // Should fit in a UDP packet
	MaxMessageSize        = math.MaxUint8
	MaxMetaKeySize        = math.MaxUint8
	MaxMetaValueSize      = math.MaxUint16
	MaxStackTracePathSize = math.MaxUint8
	MaxMetaCount          = 32
	MaxMetricCount        = 32
	MaxStackTraceCount    = 16
	MaxTagsCount          = 8
)

const (
	_0_BucketId level = iota
	_1_EntryId
	_2_Severity
	_3_Message
	_4_CategoryId
	_5_Tags
	_6_Metric
	_7_Meta
	_8_Stack_trace
	_9_TTL_Entry
	_10_TTL_Meta
	_End_Level
)

type Entry struct {
	MetaKeys        [MaxMetaCount]string
	MetaValues      [MaxMetaCount]string
	MetricKeys      [MaxMetricCount]string
	MetricValues    [MaxMetricCount]int32
	StackTracePaths [MaxStackTraceCount]string
	StackTraceLines [MaxStackTraceCount]uint16
	Tags            [MaxTagsCount]string
	Message         string
	Id              xid.ID
	BucketId        uint32
	TtlEntry        uint16
	TtlMeta         uint16
	Severity        Severity
	Level           level
	CategoryId      uint8
	TagsCount       uint8
	MetaCount       uint8
	MetricCount     uint8
	StackTraceCount uint8
}

func (e Entry) String() string {
	return e.Message
}

func (e Entry) Error() string {
	return e.Message
}

func (e *Entry) Reset() {
	e.Level = _3_Message
	e.TagsCount = 0
	e.MetricCount = 0
	e.MetaCount = 0
	e.StackTraceCount = 0
}

func (e *Entry) Encode(b []byte) (s int) {
	var i uint8
	var l level
	s += 2

	for l = 0; l <= e.Level; l++ {
		switch l {

		case _0_BucketId:
			binary.BigEndian.PutUint32(b[s:], e.BucketId)
			s += 4

		case _1_EntryId:
			s += copy(b[s:], e.Id[:])

		case _2_Severity:
			b[s] = uint8(e.Severity)
			s++

		case _3_Message:
			b[s] = uint8(len(e.Message))
			s++
			s += copy(b[s:], e.Message)

		case _4_CategoryId:
			b[s] = e.CategoryId
			s++

		case _5_Tags:
			b[s] = e.TagsCount
			s++
			for i = 0; i < e.TagsCount; i++ {
				b[s] = uint8(len(e.Tags[i]))
				s++
				s += copy(b[s:], e.Tags[i])
			}

		case _6_Metric:
			pos := s
			b[s] = e.MetricCount
			s++
			for i = 0; i < e.MetricCount; i++ {
				keyLen := len(e.StackTracePaths[i])

				if s+keyLen+5 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(keyLen)
				s++
				s += copy(b[s:], e.MetricKeys[i])
				b[s] = byte(e.MetricValues[i] >> 24)
				b[s+1] = byte(e.MetricValues[i] >> 16)
				b[s+2] = byte(e.MetricValues[i] >> 8)
				b[s+3] = byte(e.MetricValues[i])
				s += 4
			}

		case _7_Meta:
			pos := s
			b[s] = e.MetaCount
			s++
			for i = 0; i < e.MetaCount; i++ {
				keyLen := len(e.MetaKeys[i])
				valLen := len(e.MetaKeys[i])

				if s+keyLen+valLen+3 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(keyLen)
				s++
				s += copy(b[s:], e.MetaKeys[i])
				size := uint16(valLen)
				b[s] = byte(size >> 8)
				b[s+1] = byte(size)
				s += 2
				s += copy(b[s:], e.MetaValues[i])
			}

		case _8_Stack_trace:
			pos := s
			b[s] = e.StackTraceCount
			s++
			for i = 0; i < e.StackTraceCount; i++ {
				pathLen := len(e.StackTracePaths[i])

				if s+pathLen+3 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(pathLen)
				s++
				s += copy(b[s:], e.StackTracePaths[i])
				b[s] = byte(e.StackTraceLines[i] >> 8)
				b[s+1] = byte(e.StackTraceLines[i])
				s += 2
			}

		case _9_TTL_Entry:
			binary.BigEndian.PutUint16(b[s:], e.TtlEntry)
			s += 2

		case _10_TTL_Meta:
			binary.BigEndian.PutUint16(b[s:], e.TtlMeta)
			s += 2
		}
	}

	binary.BigEndian.PutUint16(b, uint16(s))

	return
}

func (e *Entry) Decode(b []byte, noCopy ...bool) (err error) {
	e.Reset()

	var unsafe bool

	if noCopy != nil && noCopy[0] {
		unsafe = true
	}

	if len(b) < 2 {
		return ErrTooShort
	}

	var s uint16
	total := binary.BigEndian.Uint16(b[s:])
	s += 2

	if uint16(len(b)) != total {
		return ErrCorruptEntry
	}

	for e.Level = 0; e.Level < _End_Level; e.Level++ {
		switch e.Level {

		case _0_BucketId:
			e.BucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		case _1_EntryId:
			if e.Id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		case _2_Severity:
			e.Severity = Severity(b[s])
			s++

		case _3_Message:
			size := uint16(b[s])
			s++
			e.Message = toString(b[s:s+size], unsafe)
			s += size

		case _4_CategoryId:
			e.CategoryId = b[s]
			s++

		case _5_Tags:
			e.TagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.TagsCount; i++ {
				size := uint16(b[s])
				s++
				e.Tags[i] = toString(b[s:s+size], unsafe)
				s += size
			}

		case _6_Metric:
			e.MetricCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.MetricCount; i++ {
				size := uint16(b[s])
				s++
				e.MetricKeys[i] = toString(b[s:s+size], unsafe)
				s += size

				e.MetricValues[i] = int32(binary.BigEndian.Uint32(b[s : s+4]))
				s += 4
			}

		case _7_Meta:
			e.MetaCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.MetaCount; i++ {
				size := uint16(b[s])
				s++
				e.MetaKeys[i] = toString(b[s:s+size], unsafe)
				s += size

				size = binary.BigEndian.Uint16(b[s : s+2])
				s += 2
				e.MetaValues[i] = toString(b[s:s+size], unsafe)
				s += size
			}

		case _8_Stack_trace:
			e.StackTraceCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.StackTraceCount; i++ {
				size := uint16(b[s])
				s++
				e.StackTracePaths[i] = toString(b[s:s+size], unsafe)
				s += size

				e.StackTraceLines[i] = binary.BigEndian.Uint16(b[s : s+2])
				s += 2
			}

		case _9_TTL_Entry:
			e.TtlEntry = binary.BigEndian.Uint16(b[s:])
			s += 2

		case _10_TTL_Meta:
			e.TtlMeta = binary.BigEndian.Uint16(b[s:])
			s += 2
		}

		if s >= total {
			break
		}
	}

	return
}

type entryWriter interface {
	writeEntry(e *Entry)
}

func (e *Entry) parseArgs(args []any) {
	for i := range args {
		switch v := args[i].(type) {

		case entryWriter:
			v.writeEntry(e)

		case func(*Entry):
			v(e)

		default:
			if e.TagsCount < 32 {
				e.Tags[e.TagsCount] = truncate(stringify(v), math.MaxUint8)
				e.TagsCount++
				e.Level = max(e.Level, 5)
			}
		}
	}
}

func (e *Entry) addStackTrace(skip int) {
	var trace [16]uintptr
	n := runtime.Callers(skip, trace[:])

	if n == 0 {
		return
	}

	frames := runtime.CallersFrames(trace[:n])
	e.StackTraceCount = uint8(n)
	e.Level = _8_Stack_trace

	for i := 0; i < n; i++ {
		frame, ok := frames.Next()
		e.StackTracePaths[i] = frame.File
		e.StackTraceLines[i] = uint16(frame.Line)

		if !ok {
			break
		}
	}
}

func toString(b []byte, unsafe bool) string {
	if unsafe {
		return bytesToString(b)
	}

	return string(b)
}

// func (e *Entry) DecodeWithoutCopy(b []byte) (err error) {
// 	if len(b) < 2 {
// 		return ErrTooShort
// 	}

// 	var s uint16
// 	total := binary.BigEndian.Uint16(b[s:])
// 	s += 2

// 	if uint16(len(b)) != total {
// 		return ErrCorruptEntry
// 	}

// 	for e.Level = 0; e.Level <= _7_Meta; e.Level++ {
// 		switch e.Level {

// 		case _0_BucketId:
// 			e.BucketId = binary.BigEndian.Uint32(b[s:])
// 			s += 4

// 		case _1_EntryId:
// 			if e.Id, err = xid.FromBytes(b[s : s+12]); err != nil {
// 				return
// 			}

// 			s += 12

// 		case _2_Severity:
// 			e.Severity = Severity(b[12])
// 			s++

// 		case _3_Message:
// 			size := uint16(b[s])
// 			s++
// 			e.Message = bytesToString(b[s : s+size])
// 			s += size

// 		case _4_Category:
// 			size := uint16(b[s])
// 			s++
// 			e.Category = bytesToString(b[s : s+size])
// 			s += size

// 		case _5_ProcId:
// 			size := uint16(b[s])
// 			s++
// 			e.ProcId = bytesToString(b[s : s+size])
// 			s += size

// 		case _6_Tags:
// 			e.TagsCount = b[s]
// 			s++
// 			var i uint8
// 			for i = 0; i < e.TagsCount; i++ {
// 				size := uint16(b[s])
// 				s++
// 				e.Tags[i] = bytesToString(b[s : s+size])
// 				s += size
// 			}

// 		case _7_Meta:
// 			e.MetaCount = b[s]
// 			s++
// 			var i uint8
// 			for i = 0; i < e.MetaCount; i++ {
// 				size := uint16(b[s])
// 				s++
// 				e.MetaKeys[i] = bytesToString(b[s : s+size])
// 				s += size

// 				size = binary.BigEndian.Uint16(b[s : s+2])
// 				s += 2
// 				e.MetaValues[i] = bytesToString(b[s : s+size])
// 				s += size
// 			}
// 		}

// 		if s >= total {
// 			break
// 		}
// 	}

// 	return
// }

// Implements encoding.BinaryMarshaler
func (e Entry) MarshalBinary() ([]byte, error) {
	var b [MaxEntrySize]byte
	s := e.Encode(b[:])
	return b[:s], nil
}

// Implements encoding.BinaryUnmarshaler
func (e *Entry) UnmarshalBinary(b []byte) error {
	return e.Decode(b)
}

func (dst *Entry) Append(src *Entry) {
	if dst.BucketId == 0 {
		dst.BucketId = src.BucketId
	}

	if dst.CategoryId == 0 {
		dst.CategoryId = src.CategoryId
	}

	if dst.TtlEntry == 0 {
		dst.TtlEntry = src.TtlEntry
	}

	if dst.TtlMeta == 0 {
		dst.TtlMeta = src.TtlMeta
	}

	if src.Level > dst.Level {
		dst.Level = src.Level
	}

	for i := range src.Tags[:src.TagsCount] {
		if dst.TagsCount >= MaxTagsCount {
			break
		}

		dst.Tags[dst.TagsCount] = src.Tags[i]
		dst.TagsCount++
	}

	for i := range src.MetaKeys[:src.MetaCount] {
		if dst.MetaCount >= MaxMetaCount {
			break
		}

		dst.MetaKeys[dst.MetaCount] = src.MetaKeys[i]
		dst.MetaValues[dst.MetaCount] = src.MetaValues[i]
		dst.MetaCount++
	}

	for i := range src.MetricKeys[:src.MetricCount] {
		if dst.MetricCount >= MaxMetricCount {
			break
		}

		dst.MetricKeys[dst.MetricCount] = src.MetricKeys[i]
		dst.MetricValues[dst.MetricCount] = src.MetricValues[i]
		dst.MetricCount++
	}
}
