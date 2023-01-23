package logger

import "testing"

func BenchmarkEntryMetric(b *testing.B) {
	var e Entry

	b.Run("CreateAndWrite", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Metric("foo", 123).writeEntry(&e)
		}
	})
}
