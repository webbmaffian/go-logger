package logger

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

type Secret [32]byte

func (s Secret) String() string {
	return hex.EncodeToString(s[:])
}

func (s Secret) MarshalJSON() ([]byte, error) {
	var b [66]byte
	b[0] = '"'
	b[65] = '"'
	hex.Encode(b[1:], s[:])

	return b[:], nil
}

func (s *Secret) Generate() (err error) {
	_, err = rand.Read(s[:])
	return
}

func (s *Secret) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		copy(s[:], v)
	case *[]byte:
		copy(s[:], *v)
	case string:
		*s = SecretFromString(v)
	case *string:
		*s = SecretFromString(*v)
	default:
		return errors.New("invalid scan type")
	}

	return nil
}

func SecretFromString(str string) (s Secret) {
	hex.Decode(s[:], stringToBytes(str))
	return
}
