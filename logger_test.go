package logger

import (
	"context"
	"testing"
	"time"
)

func BenchmarkLogMetrics(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// logger.log(DEBUG, "", nil)
		logger.Debug("Hello world")
	}

	// logger.Close()
}

func BenchmarkLog10Fields(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	now := time.Now()
	dur := time.Hour

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// logger.log(DEBUG, "", nil)
		logger.Debug("Hello world", Meta(
			"foo", "bar",
			"foo", now,
			"foo", dur,
			"foo", 123.456,
			"foo", "bar",
			"foo", now,
			"foo", dur,
			"foo", 123.456,
			"foo", []string{"bar"},
			"foo", now,
		), Category(123))
	}

	// logger.Close()
}

func BenchmarkBareboneLog(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// logger.log(DEBUG, "", nil)
		logger.Debug("")
	}

	// logger.Close()
}

func BenchmarkNewEntry(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e := logger.newEntry(DEBUG, "", nil)
		logger.queue.releaseEntry(e)
	}
}

func BenchmarkLog(b *testing.B) {
	logger := New(context.Background(), &dummyWriter{}, LoggerOptions{
		TimeNow:            FastTimeNow(context.Background()),
		StackTraceSeverity: NOTICE,
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// logger.Err("Created order %s", "12345", Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"), Meta("foo", "bar"))
		logger.Debug("Created order %s")
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
