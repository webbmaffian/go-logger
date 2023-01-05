package logger

import (
	"testing"

	"github.com/rs/xid"
)

func BenchmarkEntryEncode(b *testing.B) {
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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e.Encode(buf[:])
	}
}

func BenchmarkEntryDecode(b *testing.B) {
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
		e.Decode(buf[:size])
	}
}
