package logger

import (
	"strconv"
	"testing"
)

func BenchmarkTruncateShort(b *testing.B) {
	shortString := "foobar"

	for i := 0; i < b.N; i++ {
		_ = truncate(shortString, 255)
	}
}

func BenchmarkTruncateLong(b *testing.B) {
	longString := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque vehicula felis in lorem bibendum rutrum. Nam ultricies est sit amet ex rutrum, id blandit lacus ultricies. In lacinia convallis dui, varius ullamcorper sapien euismod non. Etiam fringilla fermentum turpis, posuere aliquet ante. Maecenas tempus odio a ipsum tincidunt ornare. Vestibulum vitae vehicula leo. Interdum et malesuada fames ac ante ipsum primis in faucibus. Mauris in finibus risus. Nulla pharetra odio ut blandit fermentum. Morbi malesuada iaculis feugiat."

	for i := 0; i < b.N; i++ {
		_ = truncate(longString, 255)
	}
}

func BenchmarkS2b(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = s2b("foobar")
	}
}

func BenchmarkStringify(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = stringify(123.123)
	}
}

func BenchmarkFormatInt(b *testing.B) {
	buf := make([]byte, 0, 32)
	b.ResetTimer()

	b.Run("FormatInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = strconv.FormatInt(123, 10)
		}
	})

	b.Run("AppendInt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf = strconv.AppendInt(buf[:0], 123, 10)
		}
	})
}
