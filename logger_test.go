package logger

import (
	"testing"

	"github.com/kpango/fastime"
	"github.com/rs/xid"
)

func BenchmarkBareboneLog(b *testing.B) {
	entryPool := NewEntryPool()
	loggerPool := LoggerPool{
		EntryPool:      entryPool,
		EntryProcessor: &dummyWriter{entryPool},
		Clock:          fastime.New(),
	}
	logger := loggerPool.Logger()
	id := xid.New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Debug("hello").Tag(id.String()).Send()
	}
	// logger.Close()
}
