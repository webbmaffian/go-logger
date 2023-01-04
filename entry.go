package logger

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/rs/xid"
)

/*
	0. EntryId
		12 byte XID
	1. Severity
		1 byte (uint8)
	2. Message
		1 byte (uint8) length (X)
		X bytes string
	3. Category
		1 byte (uint8) length (X)
		X bytes string
	4. ProcId
		1 byte (uint8) length (X)
		X bytes string
	5. Tags
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string
	6. Meta
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string key
			2 bytes (uint16) length (Y)
			Y bytes string value
	X. Metrics
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string key
			8 bytes (uint64) integer value

*/

const entrySize = math.MaxUint16 // 2 bytes

type Entry interface {
	EncodeEntry(b []byte) (s int)
	DecodeEntry(b []byte) (err error)
}

type entry struct {
	id         xid.ID
	severity   Severity
	category   string
	procId     string
	message    string
	tags       [32]string
	tagsCount  uint8
	metaKeys   [32]string
	metaValues [32]string
	metaCount  uint8
	level      int
}

func (e *entry) Read(b []byte) (n int, err error) {
	return e.encode(b), io.EOF
}

func (e *entry) encode(b []byte) (s int) {
	var i uint8

	// 0. Entry ID (XID)
	s += copy(b[:12], e.id[:])

	// 1. Severity
	b[s] = uint8(e.severity)
	s++

	for l := 2; l <= e.level; l++ {
		switch l {

		case 2: // Message
			b[s] = uint8(len(e.message))
			s++
			s += copy(b[s:], stringToBytes(e.message))

		case 3: // Category
			b[s] = uint8(len(e.category))
			s++
			s += copy(b[s:], stringToBytes(e.category))

		case 4: // Proc ID
			b[s] = uint8(len(e.procId))
			s++
			s += copy(b[s:], stringToBytes(e.procId))

		case 5: // Tags
			b[s] = e.tagsCount
			s++
			for i = 0; i < e.tagsCount; i++ {
				b[s] = uint8(len(e.tags[i]))
				s++
				s += copy(b[s:], stringToBytes(e.tags[i]))
			}

		case 6: // Meta
			pos := s
			b[s] = e.metaCount
			s++
			for i = 0; i < e.metaCount; i++ {
				keyLen := len(e.metaKeys[i])
				valLen := len(e.metaKeys[i])

				if s+keyLen+valLen+3 > entrySize {
					b[pos] = i
					break
				}

				b[s] = uint8(keyLen)
				s++
				s += copy(b[s:], stringToBytes(e.metaKeys[i]))
				size := uint16(valLen)
				b[s] = byte(size >> 8)
				b[s+1] = byte(size)
				s += 2
				s += copy(b[s:], stringToBytes(e.metaValues[i]))
			}
		}
	}

	return
}

func (e *entry) decode(b []byte) (err error) {
	var s int = 13
	total := len(b)

	if total < s {
		return errors.New("message too short")
	}

	// 0. Entry ID (XID)
	e.id, err = xid.FromBytes(b[:12])

	if err != nil {
		return
	}

	// 1. Severity
	e.severity = Severity(b[12])
	l := 2

	for s < total {
		switch l {

		case 2: // Message
			size := int(b[s])
			s++
			e.message = string(b[s : s+size])
			s += size
			l++

		case 3: // Category
			size := int(b[s])
			s++
			e.category = string(b[s : s+size])
			s += size
			l++

		case 4: // Proc ID
			size := int(b[s])
			s++
			e.procId = string(b[s : s+size])
			s += size
			l++

		case 5: // Tags
			e.tagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.tagsCount; i++ {
				size := int(b[s])
				s++
				e.tags[i] = string(b[s : s+size])
				s += size
			}
			l++

		case 6: // Meta
			e.metaCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.metaCount; i++ {
				size := int(b[s])
				s++
				e.metaKeys[i] = string(b[s : s+size])
				s += size

				size = int(binary.BigEndian.Uint16(b[s : s+2]))
				s += 2
				e.metaValues[i] = string(b[s : s+size])
				s += size
			}
			l++
		}
	}

	return
}
