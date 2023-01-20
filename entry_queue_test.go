package logger

import (
	"testing"
	"time"
)

func BenchmarkEntryQueue(b *testing.B) {
	queue := newEntryQueue(100)
	now := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e := queue.acquireEntry(now)
		queue.releaseEntry(e)
	}
}
