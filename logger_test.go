package logger

import (
	"testing"

	"github.com/kpango/fastime"
)

func BenchmarkBareboneLog(b *testing.B) {
	entryPool := NewEntryPool()
	loggerPool := LoggerPool{
		EntryPool:      entryPool,
		EntryProcessor: &dummyWriter{entryPool},
		Clock:          fastime.New(),
	}
	logger := loggerPool.Logger()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Debug("hello").Trace().Send()
	}
}
