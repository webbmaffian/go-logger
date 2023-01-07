package logger

import (
	"context"
	"testing"
	"time"
)

func BenchmarkTimeNow(b *testing.B) {
	var t func() time.Time

	t = time.Now

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = t()
	}
}

func BenchmarkFastTimeNow(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var t func() time.Time

	t = FastTimeNow(ctx)

	for i := 0; i < b.N; i++ {
		_ = t()
	}
}
