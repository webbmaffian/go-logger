package logger

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/rs/xid"
)

/*
	0. BucketId
		4 byte (uint32) integer
	1. EntryId
		12 byte XID
	2. Severity
		1 byte (uint8)
	3. Message
		1 byte (uint8) length (X)
		X bytes string
	4. Category
		1 byte (uint8) length (X)
		X bytes string
	5. ProcId
		1 byte (uint8) length (X)
		X bytes string
	6. Tags
		1 byte (uint8) count
			1 byte (uint8) length (X)
			X bytes string
	7. Meta
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

const entrySize = 65_507 // Maxiumum size of a UDP packet

type Entry struct {
	tags       [32]string
	metaKeys   [32]string
	metaValues [32]string
	category   string
	procId     string
	message    string
	id         xid.ID
	bucketId   uint32
	severity   Severity
	level      uint8
	tagsCount  uint8
	metaCount  uint8
}

func (e Entry) String() string {
	return e.message
}

func (e *Entry) Read(b []byte) (n int, err error) {
	return e.Encode(b), io.EOF
}

func (e *Entry) Encode(b []byte) (s int) {
	var i, l uint8

	// 0. Bucket ID
	binary.BigEndian.PutUint32(b[s:], e.bucketId)
	s += 4
	l++

	// 1. Entry ID (XID)
	s += copy(b[s:], e.id[:])
	l++

	// 2. Severity
	b[s] = uint8(e.severity)
	s++
	l++

	for ; l <= e.level; l++ {
		switch l {

		case 3: // Message
			b[s] = uint8(len(e.message))
			s++
			s += copy(b[s:], e.message)

		case 4: // Category
			b[s] = uint8(len(e.category))
			s++
			s += copy(b[s:], e.category)

		case 5: // Proc ID
			b[s] = uint8(len(e.procId))
			s++
			s += copy(b[s:], e.procId)

		case 6: // Tags
			b[s] = e.tagsCount
			s++
			for i = 0; i < e.tagsCount; i++ {
				b[s] = uint8(len(e.tags[i]))
				s++
				s += copy(b[s:], e.tags[i])
			}

		case 7: // Meta
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
				s += copy(b[s:], e.metaKeys[i])
				size := uint16(valLen)
				b[s] = byte(size >> 8)
				b[s+1] = byte(size)
				s += 2
				s += copy(b[s:], e.metaValues[i])
			}
		}
	}

	return
}

func (e *Entry) Decode(b []byte) (err error) {
	var s int = 17
	total := len(b)

	if total < s {
		return errors.New("message too short")
	}

	// 0. Bucket ID
	e.bucketId = binary.BigEndian.Uint32(b[:])

	// 1. Entry ID (XID)
	e.id, err = xid.FromBytes(b[:12])

	if err != nil {
		return
	}

	// 2. Severity
	e.severity = Severity(b[12])
	l := 3

	for s < total {
		switch l {

		case 3: // Message
			size := int(b[s])
			s++
			e.message = string(b[s : s+size])
			s += size
			l++

		case 4: // Category
			size := int(b[s])
			s++
			e.category = string(b[s : s+size])
			s += size
			l++

		case 5: // Proc ID
			size := int(b[s])
			s++
			e.procId = string(b[s : s+size])
			s += size
			l++

		case 6: // Tags
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

		case 7: // Meta
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
