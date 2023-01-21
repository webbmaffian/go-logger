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

func BenchmarkChannelPush(b *testing.B) {
	q := newEntryQueue(b.N)
	e := &Entry{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.putEntry(e)
	}
}
