package logger

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkLog(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{
		entryPool: sync.Pool{
			New: func() any {
				return new(entry)
			},
		},
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Debug("Created order %s", "12345", Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"))
	}

	logger.Close()
}

func BenchmarkLogTcp(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())

	// server := Server{}

	// go server.Listen(ctx)

	client, err := NewClient(ctx, ClientOptions{
		Host:   "localhost",
		Port:   4610,
		Buffer: 100,
	})

	if err != nil {
		b.Fatal(err)
	}

	logger := New(ctx, client)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Debug("Created order %s", "12345", Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"))
	}

	logger.Close()
	cancel()
}

func BenchmarkLogTcpParallell(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())

	// server := Server{}

	// go server.Listen(ctx)

	client, err := NewClient(ctx, ClientOptions{
		Host:   "localhost",
		Port:   4610,
		Buffer: 100,
	})

	if err != nil {
		b.Fatal(err)
	}

	logger := New(ctx, client)

	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			logger.Debug("Created order %s", "12345", Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"))
		}
	})

	logger.Close()
	cancel()
}
