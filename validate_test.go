package logger

import (
	"testing"

	"github.com/rs/xid"
)

func TestValidateEntryBytes(t *testing.T) {
	var buf [1024]byte

	e := Entry{
		Id:         xid.New(),
		Category:   "foobar",
		ProcId:     "barfoo",
		Message:    "lorem ipsum dolor sit amet",
		Tags:       [32]string{"foo", "bar", "baz"},
		TagsCount:  3,
		MetaKeys:   [32]string{"foo", "bar", "baz"},
		MetaValues: [32]string{"foo", "bar", "baz"},
		MetaCount:  3,
		Level:      5,
	}

	size := e.Encode(buf[:])

	if err := validateEntryBytes(buf[:size]); err != nil {
		t.Error(err)
	}
}

func BenchmarkValidateEntryBytes(b *testing.B) {
	var buf [1024]byte

	e := Entry{
		Id:         xid.New(),
		Category:   "foobar",
		ProcId:     "barfoo",
		Message:    "lorem ipsum dolor sit amet",
		Tags:       [32]string{"foo", "bar", "baz"},
		TagsCount:  3,
		MetaKeys:   [32]string{"foo", "bar", "baz"},
		MetaValues: [32]string{"foo", "bar", "baz"},
		MetaCount:  3,
		Level:      5,
	}

	size := e.Encode(buf[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validateEntryBytes(buf[:size])
	}
}
