package analyzer

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func BenchmarkLargeFile(b *testing.B) {
	d := model.Document{Format: "srt", Entries: make([]model.Entry, 50000)}
	for i := range d.Entries {
		d.Entries[i] = model.Entry{Index: i + 1, Start: int64(i) * 3000, End: int64(i)*3000 + 2000, Text: "Original subtitle sample."}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Scan(d, 42, 20)
	}
}
