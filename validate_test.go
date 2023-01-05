package logger

import (
	"testing"

	"github.com/rs/xid"
)

func TestValidateEntryBytes(t *testing.T) {
	var buf [1024]byte

	e := Entry{
		id:         xid.New(),
		category:   "foobar",
		procId:     "barfoo",
		message:    "lorem ipsum dolor sit amet",
		tags:       [32]string{"foo", "bar", "baz"},
		tagsCount:  3,
		metaKeys:   [32]string{"foo", "bar", "baz"},
		metaValues: [32]string{"foo", "bar", "baz"},
		metaCount:  3,
		level:      5,
	}

	size := e.Encode(buf[:])

	if err := validateEntryBytes(buf[:size]); err != nil {
		t.Error(err)
	}
}

func BenchmarkValidateEntryBytes(b *testing.B) {
	var buf [1024]byte

	e := Entry{
		id:         xid.New(),
		category:   "foobar",
		procId:     "barfoo",
		message:    "lorem ipsum dolor sit amet",
		tags:       [32]string{"foo", "bar", "baz"},
		tagsCount:  3,
		metaKeys:   [32]string{"foo", "bar", "baz"},
		metaValues: [32]string{"foo", "bar", "baz"},
		metaCount:  3,
		level:      5,
	}

	size := e.Encode(buf[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validateEntryBytes(buf[:size])
	}
}
