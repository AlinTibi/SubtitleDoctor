package analyzer

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func TestDiagnostics(t *testing.T) {
	d := model.Document{Format: "srt", Entries: []model.Entry{{Index: 7, Start: 1000, End: 5000, Text: "Hello"}, {Index: 2, Start: 2000, End: 3000, Text: "Same"}, {Index: 3, Start: 2000, End: 3000, Text: "Same"}, {Index: 4, Start: 7000, End: 6000, Text: "Bad duration"}, {Index: 5, Start: 8000, End: 8000, Text: ""}}}
	seen := map[string]bool{}
	for _, i := range Scan(d, 42, 20) {
		seen[i.Code] = true
	}
	for _, code := range []string{"overlap", "duplicate", "duplicate_timestamp", "numbering", "negative_duration", "zero_duration", "empty"} {
		if !seen[code] {
			t.Error("missing", code)
		}
	}
}
func TestLongNestedOverlap(t *testing.T) {
	d := model.Document{Entries: []model.Entry{{Start: 0, End: 10000, Text: "a"}, {Start: 2000, End: 3000, Text: "b"}, {Start: 4000, End: 5000, Text: "c"}}}
	n := 0
	for _, v := range Scan(d, 42, 20) {
		if v.Code == "overlap" {
			n++
		}
	}
	if n != 2 {
		t.Fatal(n)
	}
}
