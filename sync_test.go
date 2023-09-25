package logger

import (
	"sync"
	"sync/atomic"
	"testing"
)

func BenchmarkMutex(b *testing.B) {
	var mu sync.Mutex

	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkMutexWLock(b *testing.B) {
	var mu sync.RWMutex

	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkMutexRLock(b *testing.B) {
	var mu sync.RWMutex

	for i := 0; i < b.N; i++ {
		mu.RLock()
		mu.RUnlock()
	}
}

func BenchmarkAtomicPointerWrite(b *testing.B) {
	var ap atomic.Pointer[struct{}]
	p := &struct{}{}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ap.Store(p)
	}
}

func BenchmarkAtomicPointerRead(b *testing.B) {
	var ap atomic.Pointer[struct{}]
	p := &struct{}{}
	ap.Store(p)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ap.Load()
	}
}
