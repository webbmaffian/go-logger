package logger

import "testing"

func BenchmarkEntryMeta(b *testing.B) {
	var e Entry
	var metas []meta

	b.Run("Create", func(b *testing.B) {
		metas = make([]meta, b.N)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			metas[i] = Meta("foo", "bar")
		}
	})

	b.Run("WriteToEntry", func(b *testing.B) {
		l := len(metas)
		for i := 0; i < b.N; i++ {
			metas[i%l].writeEntry(&e)
			e.MetaCount = 0
		}
	})
}

func BenchmarkEntryMetaBasic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Meta("foo", "bar")
	}
}
