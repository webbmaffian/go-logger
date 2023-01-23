package logger

import (
	"context"
	"testing"

	"github.com/rs/xid"
)

func BenchmarkEntryEncode(b *testing.B) {
	var buf [1024]byte

	e := Entry{
		Id:         xid.New(),
		Message:    "lorem ipsum dolor sit amet",
		Tags:       [8]string{"foo", "bar", "baz"},
		TagsCount:  3,
		MetaKeys:   [32]string{"foo", "bar", "baz"},
		MetaValues: [32]string{"foo", "bar", "baz"},
		MetaCount:  3,
		Level:      _8_Stack_trace,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e.Encode(buf[:])
	}
}

func BenchmarkEntryDecode(b *testing.B) {
	var buf [1024]byte

	e := Entry{
		Id:         xid.New(),
		Message:    "lorem ipsum dolor sit amet",
		Tags:       [8]string{"foo", "bar", "baz"},
		TagsCount:  3,
		MetaKeys:   [32]string{"foo", "bar", "baz"},
		MetaValues: [32]string{"foo", "bar", "baz"},
		MetaCount:  3,
		Level:      _8_Stack_trace,
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

func BenchmarkXidGenerate(b *testing.B) {
	f := FastTimeNow(context.Background())
	b.ResetTimer()

	b.Run("Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = xid.New()
		}
	})

	b.Run("WithFastTime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = xid.NewWithTime(f())
		}
	})
}
