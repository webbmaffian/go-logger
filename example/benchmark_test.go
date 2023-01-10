package main

import (
	"encoding/binary"
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
