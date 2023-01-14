package logger

import (
	"encoding/binary"
	"math"
	"runtime"
	"strconv"

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
	4. Category
		1 byte (uint8) length (X)
		X bytes string
	5. Tags
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string
	6. Meta
		1 byte (uint8) count (0-32)
			1 byte (uint8) length (X)
			X bytes string key
			2 bytes (uint16) length (Y)
			Y bytes string value
		[...31]
	7. Stack trace
		1 byte (uint8) count (0-8)
			1 byte (uint8) path length (X)
			X bytes string path
			2 bytes (uint16) line number
		[...7]

*/

type level uint8

const (
	MaxEntrySize          = 65_507 // Should fit a UDP packet
	MaxMessageSize        = math.MaxUint8
	MaxCategorySize       = math.MaxUint8
	MaxMetaKeySize        = math.MaxUint8
	MaxMetaValueSize      = math.MaxUint16
	MaxStackTracePathSize = math.MaxUint8
	MaxMetaCount          = 32
	MaxStackTraceCount    = 16
	MaxTagsCount          = 8
)

const (
	_0_BucketId level = iota
	_1_EntryId
	_2_Severity
	_3_Message
	_4_Category
	_5_Tags
	_6_Meta
	_7_Stack_trace
)

type Entry struct {
	MetaKeys             [MaxMetaCount]string
	MetaValues           [MaxMetaCount]string
	StackTracePaths      [MaxStackTraceCount]string
	StackTraceRowNumbers [MaxStackTraceCount]uint16
	Tags                 [MaxTagsCount]string
	Category             string
	Message              string
	Id                   xid.ID
	BucketId             uint32
	Severity             Severity
	Level                level
	TagsCount            uint8
	MetaCount            uint8
	StackTraceCount      uint8
}

func (e Entry) String() string {
	return e.Message
}

func (e Entry) Error() string {
	return e.Message
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

		case _4_Category:
			b[s] = uint8(len(e.Category))
			s++
			s += copy(b[s:], e.Category)

		case _5_Tags:
			b[s] = e.TagsCount
			s++
			for i = 0; i < e.TagsCount; i++ {
				b[s] = uint8(len(e.Tags[i]))
				s++
				s += copy(b[s:], e.Tags[i])
			}

		case _6_Meta:
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

		case _7_Stack_trace:
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
				b[s] = byte(e.StackTraceRowNumbers[i] >> 8)
				b[s+1] = byte(e.StackTraceRowNumbers[i])
				s += 2
			}
		}
	}

	binary.BigEndian.PutUint16(b, uint16(s))

	return
}

func (e *Entry) Decode(b []byte, noCopy ...bool) (err error) {
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

	for e.Level = 0; e.Level <= _7_Stack_trace; e.Level++ {
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
			e.Severity = Severity(b[12])
			s++

		case _3_Message:
			size := uint16(b[s])
			s++
			e.Message = toString(b[s:s+size], unsafe)
			s += size

		case _4_Category:
			size := uint16(b[s])
			s++
			e.Category = toString(b[s:s+size], unsafe)
			s += size

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

		case _6_Meta:
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

		case _7_Stack_trace:
			e.StackTraceCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.StackTraceCount; i++ {
				size := uint16(b[s])
				s++
				e.StackTracePaths[i] = toString(b[s:s+size], unsafe)
				s += size

				e.StackTraceRowNumbers[i] = binary.BigEndian.Uint16(b[s : s+2])
				s += 2
			}
		}

		if s >= total {
			break
		}
	}

	return
}

func (e *Entry) parseArgs(args []any) {
	for i := range args {
		switch v := args[i].(type) {

		case Severity:
			e.Severity = v

		case Category:
			e.Category = truncate(string(v), math.MaxUint8)
			e.Level = max(e.Level, 3)

		case string:
			if e.TagsCount < 32 {
				e.Tags[e.TagsCount] = truncate(v, math.MaxUint8)
				e.TagsCount++
				e.Level = max(e.Level, 5)
			}

		case int:
			if e.TagsCount < 32 {
				e.Tags[e.TagsCount] = strconv.Itoa(v)
				e.TagsCount++
				e.Level = max(e.Level, 5)
			}

		case meta:
			if e.MetaCount < 32 {
				e.MetaKeys[e.MetaCount] = truncate(v.key, math.MaxUint8)
				e.MetaValues[e.MetaCount] = truncate(v.value, math.MaxUint16)
				e.MetaCount++
				e.Level = max(e.Level, 6)
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
	e.Level = _7_Stack_trace

	for i := 0; i < n; i++ {
		frame, ok := frames.Next()
		e.StackTracePaths[i] = frame.File
		e.StackTraceRowNumbers[i] = uint16(frame.Line)

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
