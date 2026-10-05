package repair

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/analyzer"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func TestASSDuplicateCurrentValuesAndMetadata(t *testing.T) {
	for _, format := range []string{"ass", "ssa"} {
		d := model.Document{Format: format, EventFormat: []string{"Style", "End", "Name", "Start", "Effect", "Text"}}
		entry := model.Entry{Start: 1000, End: 2000, Text: "Current", Fields: []string{"Default", "0:00:05.00", "Actor", "0:00:04.00", "fade", "Old"}}
		other := entry
		other.Fields = append([]string(nil), entry.Fields...)
		other.Fields[1], other.Fields[3], other.Fields[5] = "0:00:09.00", "0:00:08.00", "Different raw text"
		d.Entries = []model.Entry{entry, other}
		hasDuplicate := func(d model.Document) bool {
			for _, v := range analyzer.Scan(d, 42, 20) {
				if v.Code == "duplicate" {
					return true
				}
			}
			return false
		}
		if !hasDuplicate(d) {
			t.Fatal("stale raw fields hid current duplicate", format)
		}
		for _, i := range []int{0, 2, 4} {
			distinct := model.Clone(d)
			distinct.Entries[1].Fields[i] = "Distinct"
			if hasDuplicate(distinct) {
				t.Fatalf("metadata field %d ignored", i)
			}
			Fix(&distinct, Options{Duplicates: true})
			if len(distinct.Entries) != 2 {
				t.Fatal("removed distinct cue")
			}
		}
		d.Entries[1].Start = 3000
		d.Entries[1].End = 4000
		d.Entries[1].Text = "Other"
		if hasDuplicate(d) {
			t.Fatal("different current values duplicate")
		}
		d.Entries[1].Start = 1000
		d.Entries[1].End = 2000
		d.Entries[1].Text = "Current"
		Fix(&d, Options{Duplicates: true})
		if len(d.Entries) != 1 {
			t.Fatal("edit-created duplicate retained")
		}
	}
}

func TestInvalidCuesAndVTTMetadata(t *testing.T) {
	d := model.Document{Entries: []model.Entry{{Invalid: true}, {Invalid: true}}}
	Fix(&d, Options{Duplicates: true, Empty: true})
	if len(d.Entries) != 2 {
		t.Fatal("invalid empty cues removed")
	}
	d = model.Document{Format: "vtt", Entries: []model.Entry{
		{Start: 1000, End: 2000, Text: "Same", Fields: []string{"id1", "align:start"}},
		{Start: 1000, End: 2000, Text: "Same", Fields: []string{"id2", "align:end"}},
	}}
	Fix(&d, Options{Duplicates: true})
	if len(d.Entries) != 2 {
		t.Fatal("VTT metadata removed")
	}
}
