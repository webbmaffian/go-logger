package logger

import (
	"encoding/binary"

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
	Tags       [32]string
	MetaKeys   [32]string
	MetaValues [32]string
	Category   string
	ProcId     string
	Message    string
	Id         xid.ID
	BucketId   uint32
	Severity   Severity
	Level      level
	TagsCount  uint8
	MetaCount  uint8
}

func (e Entry) String() string {
	return e.Message
}

// func (e *Entry) Read(b []byte) (n int, err error) {
// 	return e.Encode(b), io.EOF
// }

func (e *Entry) Encode(b []byte) (s int) {
	var i uint8
	var l level

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

		case _5_ProcId:
			b[s] = uint8(len(e.ProcId))
			s++
			s += copy(b[s:], e.ProcId)

		case _6_Tags:
			b[s] = e.TagsCount
			s++
			for i = 0; i < e.TagsCount; i++ {
				b[s] = uint8(len(e.Tags[i]))
				s++
				s += copy(b[s:], e.Tags[i])
			}

		case _7_Meta:
			pos := s
			b[s] = e.MetaCount
			s++
			for i = 0; i < e.MetaCount; i++ {
				keyLen := len(e.MetaKeys[i])
				valLen := len(e.MetaKeys[i])

				if s+keyLen+valLen+3 > entrySize {
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
		}
	}

	return
}

func (e *Entry) Decode(b []byte) (err error) {
	var s int
	total := len(b)

	for e.Level = 0; e.Level <= 7; e.Level++ {
		switch e.Level {

		case _0_BucketId: // Bucket ID
			e.BucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		case _1_EntryId: // Entry ID (XID)
			if e.Id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		case _2_Severity: // Severity
			e.Severity = Severity(b[12])
			s++

		case _3_Message: // Message
			size := int(b[s])
			s++
			e.Message = string(b[s : s+size])
			s += size

		case _4_Category: // Category
			size := int(b[s])
			s++
			e.Category = string(b[s : s+size])
			s += size

		case _5_ProcId: // Proc ID
			size := int(b[s])
			s++
			e.ProcId = string(b[s : s+size])
			s += size

		case _6_Tags: // Tags
			e.TagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.TagsCount; i++ {
				size := int(b[s])
				s++
				e.Tags[i] = string(b[s : s+size])
				s += size
			}

		case _7_Meta: // Meta
			e.MetaCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.MetaCount; i++ {
				size := int(b[s])
				s++
				e.MetaKeys[i] = string(b[s : s+size])
				s += size

				size = int(binary.BigEndian.Uint16(b[s : s+2]))
				s += 2
				e.MetaValues[i] = string(b[s : s+size])
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

	for e.Level = 0; e.Level <= 7; e.Level++ {
		switch e.Level {

		case 0: // Bucket ID
			e.BucketId = binary.BigEndian.Uint32(b[s:])
			s += 4

		case 1: // Entry ID (XID)
			if e.Id, err = xid.FromBytes(b[s : s+12]); err != nil {
				return
			}

			s += 12

		case 2: // Severity
			e.Severity = Severity(b[12])
			s++

		case 3: // Message
			size := int(b[s])
			s++
			e.Message = bytesToString(b[s : s+size])
			s += size

		case 4: // Category
			size := int(b[s])
			s++
			e.Category = bytesToString(b[s : s+size])
			s += size

		case 5: // Proc ID
			size := int(b[s])
			s++
			e.ProcId = bytesToString(b[s : s+size])
			s += size

		case 6: // Tags
			e.TagsCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.TagsCount; i++ {
				size := int(b[s])
				s++
				e.Tags[i] = bytesToString(b[s : s+size])
				s += size
			}

		case 7: // Meta
			e.MetaCount = b[s]
			s++
			var i uint8
			for i = 0; i < e.MetaCount; i++ {
				size := int(b[s])
				s++
				e.MetaKeys[i] = bytesToString(b[s : s+size])
				s += size

				size = int(binary.BigEndian.Uint16(b[s : s+2]))
				s += 2
				e.MetaValues[i] = bytesToString(b[s : s+size])
				s += size
			}
		}

		if s >= total {
			break
		}
	}

	return
}

// Implements encoding.BinaryMarshaler
func (e Entry) MarshalBinary() ([]byte, error) {
	var b [entrySize]byte
	s := e.Encode(b[:])
	return b[:s], nil
}

// Implements encoding.BinaryUnmarshaler
func (e *Entry) UnmarshalBinary(b []byte) error {
	return e.Decode(b)
}
