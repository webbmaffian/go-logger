package logger

import (
	"encoding/binary"
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

type level uint8

const (
	_0_BucketId level = iota
	_1_EntryId
	_2_Severity
	_3_Message
	_4_Category
	_5_ProcId
	_6_Tags
	_7_Meta
)

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
	level      level
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
	var i uint8
	var l level

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

		case _4_Category:
			b[s] = uint8(len(e.category))
			s++
			s += copy(b[s:], e.category)

		case _5_ProcId:
			b[s] = uint8(len(e.procId))
			s++
			s += copy(b[s:], e.procId)

		case _6_Tags:
			b[s] = e.tagsCount
			s++
			for i = 0; i < e.tagsCount; i++ {
				b[s] = uint8(len(e.tags[i]))
				s++
				s += copy(b[s:], e.tags[i])
			}

		case _7_Meta:
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
	var s int
	total := len(b)

	for e.level = 0; e.level <= 7; e.level++ {
		switch e.level {

		case _0_BucketId: // Bucket ID
			e.bucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		case _1_EntryId: // Entry ID (XID)
			if e.id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		case _2_Severity: // Severity
			e.severity = Severity(b[12])
			s++

		case _3_Message: // Message
			size := int(b[s])
			s++
			e.message = string(b[s : s+size])
			s += size

		case _4_Category: // Category
			size := int(b[s])
			s++
			e.category = string(b[s : s+size])
			s += size

		case _5_ProcId: // Proc ID
			size := int(b[s])
			s++
			e.procId = string(b[s : s+size])
			s += size

		case _6_Tags: // Tags
			e.tagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.tagsCount; i++ {
				size := int(b[s])
				s++
				e.tags[i] = string(b[s : s+size])
				s += size
			}

		case _7_Meta: // Meta
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
		}

		if s >= total {
			break
		}
	}

	return
}

func (e *Entry) DecodeWithoutCopy(b []byte) (err error) {
	var s int
	total := len(b)

	for e.level = 0; e.level <= 7; e.level++ {
		switch e.level {

		case 0: // Bucket ID
			e.bucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		case 1: // Entry ID (XID)
			if e.id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		case 2: // Severity
			e.severity = Severity(b[12])
			s++

		case 3: // Message
			size := int(b[s])
			s++
			e.message = bytesToString(b[s : s+size])
			s += size

		case 4: // Category
			size := int(b[s])
			s++
			e.category = bytesToString(b[s : s+size])
			s += size

		case 5: // Proc ID
			size := int(b[s])
			s++
			e.procId = bytesToString(b[s : s+size])
			s += size

		case 6: // Tags
			e.tagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.tagsCount; i++ {
				size := int(b[s])
				s++
				e.tags[i] = bytesToString(b[s : s+size])
				s += size
			}

		case 7: // Meta
			e.metaCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.metaCount; i++ {
				size := int(b[s])
				s++
				e.metaKeys[i] = bytesToString(b[s : s+size])
				s += size

				size = int(binary.BigEndian.Uint16(b[s : s+2]))
				s += 2
				e.metaValues[i] = bytesToString(b[s : s+size])
				s += size
			}
		}

		if s >= total {
			break
		}
	}

	return
}
