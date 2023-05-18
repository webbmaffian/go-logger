package logger

import (
	"testing"

	"github.com/kpango/fastime"
)

func BenchmarkBareboneLog(b *testing.B) {
	pool, err := NewPool(&dummyWriter{clock: fastime.New()})

	if err != nil {
		b.Fatal(err)
	}

	logger := pool.Logger()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Debug("hello").Trace().Send()
	}
}
