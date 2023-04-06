package logger

import (
	"testing"

	"github.com/rs/xid"
)

func BenchmarkEntryEncode(b *testing.B) {
	var buf [1024]byte

	e := Entry{
		id:         xid.New(),
		message:    "lorem ipsum dolor sit amet",
		tags:       [8]string{"foo", "bar", "baz"},
		tagsCount:  3,
		metaKeys:   [32]string{"foo", "bar", "baz"},
		metaValues: [32]string{"foo", "bar", "baz"},
		metaCount:  3,
		level:      _8_Stack_trace,
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
		message:    "lorem ipsum dolor sit amet",
		tags:       [8]string{"foo", "bar", "baz"},
		tagsCount:  3,
		metaKeys:   [32]string{"foo", "bar", "baz"},
		metaValues: [32]string{"foo", "bar", "baz"},
		metaCount:  3,
		level:      _8_Stack_trace,
	}

	size := e.Encode(buf[:])

	b.ResetTimer()

	b.Run("copy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			e.Decode(buf[:size])
		}
	})

	b.Run("no_copy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			e.Decode(buf[:size], true)
		}
	})
}

func BenchmarkEntryXID(b *testing.B) {
	var e Entry

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var id xid.ID
		e.Id(id)
	}
}
