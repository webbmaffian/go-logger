package main

import (
	"encoding/binary"
	"runtime"
	"testing"
)

func BenchmarkUvarint(b *testing.B) {
	var data [8]byte
	data[1] = 1

	for i := 0; i < b.N; i++ {
		binary.PutUvarint(data[:], 123)
		x, y := binary.Uvarint(data[:])
		b.Errorf("%d, %d", x, y)
	}
}

func BenchmarkStackTrace(b *testing.B) {
	var trace [16]uintptr
	var n int
	var frames *runtime.Frames
	b.ResetTimer()

	b.Run("Callers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			n = runtime.Callers(0, trace[:])
		}
	})

	b.Run("CallersFrames", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			frames = runtime.CallersFrames(trace[:n])
		}
	})

	b.Run("CallersFramesIterate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = frames.Next()
		}
	})
}
