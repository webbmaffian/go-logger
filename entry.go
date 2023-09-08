package logger

import (
	"context"
	"encoding"
	"encoding/binary"
	"math"
	"runtime"
	"strings"
	"time"

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
	MaxTagSize            = math.MaxUint8
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

// Entry implements these interfaces
var (
	_ stringer                   = Entry{}
	_ error                      = Entry{}
	_ encoding.BinaryMarshaler   = Entry{}
	_ encoding.BinaryUnmarshaler = (*Entry)(nil)
)

type Entry struct {
	metaKeys        [MaxMetaCount]string
	metaValues      [MaxMetaCount]string
	metricKeys      [MaxMetricCount]string
	metricValues    [MaxMetricCount]int32
	stackTracePaths [MaxStackTraceCount]string
	stackTraceLines [MaxStackTraceCount]uint16
	tags            [MaxTagsCount]string
	message         string
	id              xid.ID
	logger          *Logger
	bucketId        uint32
	ttlEntry        uint16
	ttlMeta         uint16
	severity        Severity
	level           level
	categoryId      uint8
	tagsCount       uint8
	metaCount       uint8
	metricCount     uint8
	stackTraceCount uint8
}

var nilId xid.ID

// Implements stringer interface
func (e Entry) String() string {
	var (
		builder    strings.Builder
		valueIndex uint8
		i          uint8
	)

	for i = 0; i < e.tagsCount; i++ {
		if e.message[i] == '%' && int(i+1) < len(e.message) {
			if e.message[i+1] == '%' {
				builder.WriteByte('%')
				i++ // Skip the second '%'
			} else {
				if valueIndex < e.tagsCount {
					builder.WriteString(e.tags[valueIndex])
					valueIndex++
					i++ // Skip the placeholder
				} else {
					builder.WriteByte('%')
				}
			}
		} else {
			builder.WriteByte(e.message[i])
		}
	}

	return builder.String()
}

// Implements error interface
func (e Entry) Error() string {
	return e.String()
}

// Reset the entry as if it was fresh from the pool
func (e *Entry) Reset() {
	e.id = nilId
	e.logger = nil
	e.level = _3_Message
	e.tagsCount = 0
	e.metricCount = 0
	e.metaCount = 0
	e.stackTraceCount = 0
	e.ttlEntry = 0
	e.ttlMeta = 0
}

// Encodes the entry to a binary representation into b. If b isn't
// large enought we will panic. Returns number of bytes written.
func (e *Entry) Encode(b []byte) (s int) {
	var i uint8
	var l level

	// Reserve two bytes for the size annotation
	s += 2

	for l = 0; l <= e.level; l++ {
		switch l {

		case _0_BucketId:
			binary.BigEndian.PutUint32(b[s:], e.bucketId)
			s += 4

		case _1_EntryId:
			s += copy(b[s:], e.id[:])

		case _2_Severity:
			b[s] = uint8(e.severity)
			s++

		case _3_Message:
			b[s] = uint8(len(e.message))
			s++
			s += copy(b[s:], e.message)

		case _4_CategoryId:
			b[s] = e.categoryId
			s++

		case _5_Tags:
			b[s] = e.tagsCount
			s++
			for i = 0; i < e.tagsCount; i++ {
				b[s] = uint8(len(e.tags[i]))
				s++
				s += copy(b[s:], e.tags[i])
			}

		case _6_Metric:
			pos := s
			b[s] = e.metricCount
			s++
			for i = 0; i < e.metricCount; i++ {
				keyLen := len(e.stackTracePaths[i])

				if s+keyLen+5 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(keyLen)
				s++
				s += copy(b[s:], e.metricKeys[i])
				b[s] = byte(e.metricValues[i] >> 24)
				b[s+1] = byte(e.metricValues[i] >> 16)
				b[s+2] = byte(e.metricValues[i] >> 8)
				b[s+3] = byte(e.metricValues[i])
				s += 4
			}

		case _7_Meta:
			pos := s
			b[s] = e.metaCount
			s++
			for i = 0; i < e.metaCount; i++ {
				keyLen := len(e.metaKeys[i])
				valLen := len(e.metaValues[i])

				if s+keyLen+valLen+3 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(keyLen)
				s++
				s += copy(b[s:], e.metaKeys[i])
				size := uint16(valLen)
				b[s] = byte(size >> 8)
				b[s+1] = byte(size)
				s += 2
				s += copy(b[s:], e.metaValues[i])
			}

		case _8_Stack_trace:
			pos := s
			b[s] = e.stackTraceCount
			s++
			for i = 0; i < e.stackTraceCount; i++ {
				pathLen := len(e.stackTracePaths[i])

				if s+pathLen+3 > MaxEntrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(pathLen)
				s++
				s += copy(b[s:], e.stackTracePaths[i])
				b[s] = byte(e.stackTraceLines[i] >> 8)
				b[s+1] = byte(e.stackTraceLines[i])
				s += 2
			}

		case _9_TTL_Entry:
			binary.BigEndian.PutUint16(b[s:], e.ttlEntry)
			s += 2

		case _10_TTL_Meta:
			binary.BigEndian.PutUint16(b[s:], e.ttlMeta)
			s += 2
		}
	}

	binary.BigEndian.PutUint16(b, uint16(s))

	return
}

// Decodes a binary representation into entry, with an option to reference to the
// byte slice directly instead of doing any copy.
func (e *Entry) Decode(b []byte, noCopy ...bool) (err error) {
	e.Reset()

	var unsafe bool

	if noCopy != nil && noCopy[0] {
		unsafe = true
	}

	// An entry must contain at least size annotation (2 bytes), bucket ID (4 bytes) and entry ID (12 bytes)
	if len(b) < 18 {
		return ErrTooShort
	}

	var s uint16

	// Existence of size annotation is already ensured. And this can't overflow as the annotation
	// is an uint16, which can't store an integer larger than 2^16, which is the max length of an entry.
	total := binary.BigEndian.Uint16(b[s:])
	s += 2

	if uint16(len(b)) != total {
		return ErrCorruptEntry
	}

loop:
	for e.level = 0; e.level < _End_Level; e.level++ {
		switch e.level {

		// Existence of bucket ID is already ensured
		case _0_BucketId:
			e.bucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		// Existence of entry ID is already ensured
		case _1_EntryId:
			if e.id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		// Only one byte, can't be out of range
		case _2_Severity:
			e.severity = Severity(b[s])
			s++

		// Message length is dynamic and must not be out of range
		case _3_Message:
			size := uint16(b[s])
			s++

			// Out of range?
			if s+size > total {
				break loop
			}

			e.message = toString(b[s:s+size], unsafe)
			s += size

		// Only one byte, can't be out of range
		case _4_CategoryId:
			e.categoryId = b[s]
			s++

		// Tags count and length are dynamic and must not be out of range
		case _5_Tags:
			e.tagsCount = b[s]
			s++

			// Out of range?
			if e.tagsCount > MaxTagsCount {
				break loop
			}

			var i uint8
			for i = 0; i < e.tagsCount; i++ {

				// Out of range?
				if s >= total {
					break loop
				}

				size := uint16(b[s])
				s++

				// Out of range?
				if s+size > total {
					break loop
				}

				e.tags[i] = toString(b[s:s+size], unsafe)
				s += size
			}

		// Metric count and length are dynamic and must not be out of range
		case _6_Metric:
			e.metricCount = b[s]
			s++

			// Out of range?
			if e.metricCount > MaxMetricCount {
				break loop
			}

			var i uint8
			for i = 0; i < e.metricCount; i++ {

				// Out of range?
				if s >= total {
					break loop
				}

				size := uint16(b[s])
				s++

				// Out of range?
				if s+size+4 > total {
					break loop
				}

				e.metricKeys[i] = toString(b[s:s+size], unsafe)
				s += size

				e.metricValues[i] = int32(binary.BigEndian.Uint32(b[s : s+4]))
				s += 4
			}

		// Meta count and length are dynamic and must not be out of range
		case _7_Meta:
			e.metaCount = b[s]
			s++

			// Out of range?
			if e.metaCount > MaxMetaCount {
				break loop
			}

			var i uint8
			for i = 0; i < e.metaCount; i++ {

				// Out of range?
				if s >= total {
					break loop
				}

				size := uint16(b[s])
				s++

				// Out of range?
				if s+size+2 >= total {
					break loop
				}

				e.metaKeys[i] = toString(b[s:s+size], unsafe)
				s += size

				size = binary.BigEndian.Uint16(b[s : s+2])
				s += 2

				// Out of range?
				if s+size > total {
					break loop
				}

				e.metaValues[i] = toString(b[s:s+size], unsafe)
				s += size
			}

		// Stack trace count and length are dynamic and must not be out of range
		case _8_Stack_trace:
			e.stackTraceCount = b[s]
			s++

			// Out of range?
			if e.stackTraceCount > MaxStackTraceCount {
				break loop
			}

			var i uint8
			for i = 0; i < e.stackTraceCount; i++ {

				// Out of range?
				if s >= total {
					break loop
				}

				size := uint16(b[s])
				s++

				// Out of range?
				if s+size+2 > total {
					break loop
				}

				e.stackTracePaths[i] = toString(b[s:s+size], unsafe)
				s += size

				e.stackTraceLines[i] = binary.BigEndian.Uint16(b[s : s+2])
				s += 2
			}

		// TTL is always 2 bytes and must not be out of range
		case _9_TTL_Entry:

			// Out of range?
			if s+2 > total {
				break loop
			}

			e.ttlEntry = binary.BigEndian.Uint16(b[s:])
			s += 2

		// TTL is always 2 bytes and must not be out of range
		case _10_TTL_Meta:
			e.ttlMeta = binary.BigEndian.Uint16(b[s:])
			s += 2
		}

		if s >= total {
			break loop
		}
	}

	if s != total {
		return ErrCorruptEntry
	}

	return
}

func (e *Entry) addStackTrace(skip int) {
	var trace [16]uintptr
	n := runtime.Callers(skip, trace[:])

	if n == 0 {
		return
	}

	frames := runtime.CallersFrames(trace[:n])
	e.stackTraceCount = uint8(n)
	e.level = _8_Stack_trace

	for i := 0; i < n; i++ {
		frame, ok := frames.Next()
		e.stackTracePaths[i] = frame.File
		e.stackTraceLines[i] = uint16(frame.Line)

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

// Sets the bucket ID of the entry. Chainable.
func (e *Entry) Bucket(bucketId uint32) *Entry {
	e.bucketId = bucketId
	return e
}

// Sets the ID of the entry. Chainable.
func (e *Entry) Id(id xid.ID) *Entry {
	e.id = id
	return e
}

// Sets the timestamp of the entry by generating a new ID based on the timestamp. Chainable.
func (e *Entry) Time(t time.Time) *Entry {
	e.id = xid.NewWithTime(t)
	return e
}

// Sets the message of the entry. Chainable.
func (e *Entry) Msg(msg string) *Entry {
	e.message = msg
	return e
}

// Sets the severity of the entry. Chainable.
func (e *Entry) Sev(severity Severity) *Entry {
	e.severity = severity
	return e
}

// Sets the category ID of the entry. Chainable.
func (e *Entry) Cat(categoryId uint8) *Entry {
	e.incLevel(_4_CategoryId)
	e.categoryId = categoryId
	return e
}

// Appends tags to the entry. Stops if the entry's number of tags exceeds `MaxTagsCount`. Chainable.
func (e *Entry) Tag(tags ...any) *Entry {
	e.incLevel(_5_Tags)

	for i := range tags {
		if e.tagsCount >= MaxTagsCount {
			break
		}

		if tags[i] == "" {
			continue
		}

		e.tags[e.tagsCount] = truncate(stringify(tags[i]), MaxTagSize)
		e.tagsCount++
	}

	return e
}

// Prepends tags to the entry and removes any tags that overflow `MaxTagsCount`. Chainable.
func (e *Entry) PrependTag(tag ...any) *Entry {
	e.incLevel(_5_Tags)

	if e.tagsCount == 0 || len(tag) >= MaxTagsCount {
		e.tagsCount = 0
		return e.Tag(tag...)
	}

	if tag != nil {
		copy(e.tags[len(tag):], e.tags[:e.tagsCount])

		for i := range tag {
			e.tags[i] = truncate(stringify(tag[i]), MaxTagSize)

			if e.tagsCount < MaxTagsCount {
				e.tagsCount++
			}
		}
	}

	return e
}

func (e *Entry) MetaBlob(value any) *Entry {
	return e.Meta("_", value)
}

func (e *Entry) Meta(key string, value any) *Entry {
	e.incLevel(_7_Meta)

	if e.metaCount >= MaxMetaCount {
		return e
	}

	if key == "" || value == "" {
		return e
	}

	e.metaKeys[e.metaCount] = truncate(key, MaxMetaKeySize)
	e.metaValues[e.metaCount] = truncate(stringify(value), MaxMetaValueSize)
	e.metaCount++

	return e
}

func (e *Entry) Metric(key string, value int32) *Entry {
	e.incLevel(_6_Metric)

	if e.metricCount >= MaxMetricCount {
		return e
	}

	if key == "" {
		return e
	}

	e.metricKeys[e.metricCount] = truncate(key, MaxMetaKeySize)
	e.metricValues[e.metricCount] = value
	e.metricCount++

	return e
}

// Sets the stack trace to the current line of code. This operation is expensive compared
// to any other method. You can optionally skip levels. Chainable.
func (e *Entry) Trace(skipLevels ...int) *Entry {
	e.incLevel(_8_Stack_trace)

	if skipLevels != nil {
		e.addStackTrace(1 + skipLevels[0])
	} else {
		e.addStackTrace(1)
	}

	return e
}

// Appens to the stack trace manually - this should most likely not be used unless you want to
// load an entry from an external source, e.g. database. Chainable.
func (e *Entry) ManualTrace(path string, line uint16) *Entry {
	e.incLevel(_8_Stack_trace)

	if e.stackTraceCount < MaxStackTraceCount {
		e.stackTracePaths[e.stackTraceCount] = path
		e.stackTraceLines[e.stackTraceCount] = line
		e.stackTraceCount++
	}

	return e
}

func (e *Entry) TTL(days uint16) *Entry {
	e.incLevel(_9_TTL_Entry)
	e.ttlEntry = days
	return e
}

func (e *Entry) MetaTTL(days uint16) *Entry {
	e.incLevel(_10_TTL_Meta)
	e.ttlMeta = days
	return e
}

// Sends the entry to the log and returns its unique ID.
func (e *Entry) Send() (id xid.ID) {
	id = e.id

	// Any tags, meta and metrics are appended from the logger in ths stage
	if e.logger != nil {
		for i := range e.logger.tags {
			if e.tagsCount > MaxTagsCount {
				break
			}

			e.tags[e.tagsCount] = e.logger.tags[i]
			e.tagsCount++
		}

		for i := range e.logger.metaKeys {
			if e.metaCount >= MaxMetaCount {
				break
			}

			e.metaKeys[e.metaCount] = e.logger.metaKeys[i]
			e.metaValues[e.metaCount] = e.logger.metaValues[i]
			e.metaCount++
		}

		for i := range e.logger.metricKeys {
			if e.metricCount >= MaxMetricCount {
				break
			}

			e.metricKeys[e.metricCount] = e.logger.metricKeys[i]
			e.metricValues[e.metricCount] = e.logger.metricValues[i]
			e.metricCount++
		}

		e.logger.pool.client.ProcessEntry(context.Background(), e)
	}

	return
}

// Returns a readable interface of the entry.
func (e *Entry) Read() entryReader {
	return entryReader{e}
}

// Releases the entry back to the pool. Any usage of the entry afterwards might panic.
func (e *Entry) Drop() {
	if e.logger != nil {
		e.logger.pool.ReleaseEntry(e)
	}
}

func (e *Entry) incLevel(lvl level) {
	e.level = max(e.level, lvl)
}
