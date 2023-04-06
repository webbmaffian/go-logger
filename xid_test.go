package logger

import (
	"testing"

	"github.com/rs/xid"
)

func BenchmarkXidFromString(b *testing.B) {
	var id xid.ID
	s := "9m4e2mr0ui3e8a215n4g"
	sb := []byte(s)

	b.Run("FromString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			id, _ = xid.FromString(s)
		}
	})

	b.Run("UnmarshalText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = id.UnmarshalText(sb)
		}
	})
}
