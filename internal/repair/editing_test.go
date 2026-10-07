package repair

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"reflect"
	"testing"
)

func TestNormalMerge(t *testing.T) {
	d := model.Document{Format: "srt", LineEnding: "CRLF", Entries: []model.Entry{{Start: 100, End: 1000, Text: "<i>Bună 🙂</i>"}, {Start: 1000, End: 2000, Text: "<b>lume</b>"}}}
	if err := EditCues(&d, []int{0, 1}, "merge"); err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 1 || d.Entries[0].Start != 100 || d.Entries[0].End != 2000 || d.Entries[0].Text != "<i>Bună 🙂</i>\n<b>lume</b>" || d.LineEnding != "CRLF" {
		t.Fatal(d)
	}
}
func TestMergeRefusalIsTransactional(t *testing.T) {
	for _, tc := range []struct {
		name          string
		format        string
		first, second model.Entry
	}{
		{"gap", "srt", model.Entry{Start: 0, End: 100, Text: "one"}, model.Entry{Start: 101, End: 200, Text: "two"}},
		{"overlap", "srt", model.Entry{Start: 0, End: 100, Text: "one"}, model.Entry{Start: 99, End: 200, Text: "two"}},
		{"empty", "srt", model.Entry{Start: 0, End: 100, Text: ""}, model.Entry{Start: 100, End: 200, Text: "two"}},
		{"invalid", "srt", model.Entry{Start: 0, End: 100, Text: "one", Invalid: true}, model.Entry{Start: 100, End: 200, Text: "two"}},
		{"VTT settings", "vtt", model.Entry{Start: 0, End: 100, Text: "one", Fields: []string{"", "align:start"}}, model.Entry{Start: 100, End: 200, Text: "two", Fields: []string{"", "align:end"}}},
		{"identifier", "vtt", model.Entry{Start: 0, End: 100, Text: "one", Fields: []string{"cue1", ""}}, model.Entry{Start: 100, End: 200, Text: "two", Fields: []string{"cue2", ""}}},
		{"ASS style", "ass", model.Entry{Start: 0, End: 100, Text: "one", Fields: []string{"0", "100", "Default", "one"}}, model.Entry{Start: 100, End: 200, Text: "two", Fields: []string{"100", "200", "Other", "two"}}},
		{"ASS override", "ass", model.Entry{Start: 0, End: 100, Text: `{\i1}one`, Fields: []string{"0", "100", "Default", "one"}}, model.Entry{Start: 100, End: 200, Text: "two", Fields: []string{"100", "200", "Default", "two"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := model.Document{Format: tc.format, EventFormat: []string{"Start", "End", "Style", "Text"}, Entries: []model.Entry{tc.first, tc.second}}
			before := model.Clone(d)
			if err := EditCues(&d, []int{0, 1}, "merge"); err == nil || !reflect.DeepEqual(d, before) {
				t.Fatal(d, err)
			}
		})
	}
}
func TestSplitTimingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, format string
		start, end   int64
		ok           bool
	}{
		{"millisecond", "srt", 0, 2, true}, {"odd", "srt", 1, 4, true}, {"too short", "srt", 0, 1, false}, {"negative", "srt", -1, 100, false}, {"reversed", "srt", 2, 1, false}, {"ASS aligned", "ass", 10, 40, true}, {"ASS short", "ass", 10, 20, false}, {"ASS unaligned", "ass", 1, 25, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := model.Document{Format: tc.format, Entries: []model.Entry{{Start: tc.start, End: tc.end, Text: "<i>one two</i>"}}}
			before := model.Clone(d)
			err := EditCues(&d, []int{0}, "split")
			if (err == nil) != tc.ok {
				t.Fatal(err)
			}
			if !tc.ok && !reflect.DeepEqual(d, before) {
				t.Fatal("changed draft")
			}
			if tc.ok && (d.Entries[0].End != d.Entries[1].Start || d.Entries[0].End <= tc.start || d.Entries[1].Start >= tc.end) {
				t.Fatal(d)
			}
		})
	}
}
func TestMergeSameASSMetadataUsesCurrentValues(t *testing.T) {
	d := model.Document{Format: "ass", EventFormat: []string{"Start", "End", "Style", "Text"}, Entries: []model.Entry{{Start: 0, End: 100, Text: "one", Fields: []string{"old1", "old2", "Default", "stale1"}}, {Start: 100, End: 200, Text: "two", Fields: []string{"old3", "old4", "Default", "stale2"}}}}
	if err := EditCues(&d, []int{0, 1}, "merge"); err != nil {
		t.Fatal(err)
	}
}

func TestMergeImplicitVoiceAndKaraokeRefused(t *testing.T) {
	for _, text := range []string{"<v Speaker>one", "one <00:00.050>two"} {
		d := model.Document{Format: "vtt", Entries: []model.Entry{{Start: 0, End: 100, Text: text}, {Start: 100, End: 200, Text: "plain dialogue"}}}
		before := model.Clone(d)
		if err := EditCues(&d, []int{0, 1}, "merge"); err == nil || !reflect.DeepEqual(d, before) {
			t.Fatal(text, err)
		}
	}
}

func TestVTTSplitKeepsSettings(t *testing.T) {
	d := model.Document{Format: "vtt", Entries: []model.Entry{{Start: 0, End: 100, Text: "one two", Fields: []string{"", "align:start position:10%"}}}}
	if err := EditCues(&d, []int{0}, "split"); err != nil {
		t.Fatal(err)
	}
	for _, e := range d.Entries {
		if !reflect.DeepEqual(e.Fields, []string{"", "align:start position:10%"}) {
			t.Fatal(e)
		}
	}
}
