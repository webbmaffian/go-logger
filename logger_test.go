package logger

import (
	"context"
	"testing"
)

func BenchmarkLog(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow: FastTimeNow(context.Background()),
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

	client := NewClient(ctx, &ClientTCP{
		Address: "localhost:4610",
	})

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

	client := NewClient(ctx, &ClientTCP{
		Address: "localhost:4610",
	})

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
