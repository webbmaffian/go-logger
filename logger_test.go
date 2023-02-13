package logger

import (
	"context"
	"testing"
)

func BenchmarkBareboneLog(b *testing.B) {
	pool := NewEntryPool()
	logger := New(context.Background(), &dummyWriter{pool}, pool, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// logger.log(DEBUG, "", nil)
		logger.Debug("hello", Meta("mja", "mja"), Meta("mja", "mja"))
	}

	// logger.Close()
}
