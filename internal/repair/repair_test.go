package repair

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func TestRepair(t *testing.T) {
	d := model.Document{Entries: []model.Entry{{Index: 9, Start: 2000, End: 2000, Text: "  Hello   world "}, {Index: 3, Start: 1000, End: 2000, Text: "One"}, {Index: 3, Start: 1000, End: 2000, Text: "One"}, {Index: 8, Start: 9000, End: 10000, Text: " "}}}
	Fix(&d, Options{Renumber: true, Empty: true, Duplicates: true, Order: true, Duration: true, Spacing: true})
	if len(d.Entries) != 2 || d.Entries[0].Index != 1 || d.Entries[1].Index != 2 || d.Entries[1].End != 3000 || d.Entries[1].Text != "Hello world" {
		t.Fatal(d)
	}
}

func TestDuplicatesPreserveCueMetadata(t *testing.T) {
	d := model.Document{Entries: []model.Entry{
		{Start: 1000, End: 2000, Text: "Hello", Fields: []string{"Top"}},
		{Start: 1000, End: 2000, Text: "Hello", Fields: []string{"Bottom"}},
		{Start: 1000, End: 2000, Text: "Hello", Fields: []string{"Top"}},
		{Text: "Unreadable", Invalid: true},
		{Text: "Unreadable", Invalid: true},
	}}
	Fix(&d, Options{Duplicates: true})
	if len(d.Entries) != 4 || d.Entries[1].Fields[0] != "Bottom" {
		t.Fatalf("distinct metadata or invalid cues were removed: %+v", d.Entries)
	}
}
func TestReplace(t *testing.T) {
	for _, tc := range []struct {
		o    Operation
		want string
		n    int
	}{{Operation{Find: "cat", Replace: "dog"}, "dog dog sdogter", 3}, {Operation{Find: "cat", Replace: "$1", WholeWord: true}, "$1 $1 scatter", 2}, {Operation{Find: "cat", Replace: "dog", CaseSensitive: true}, "Cat dog sdogter", 2}, {Operation{Find: `(C|c)at`, Replace: "${1}ow", Regex: true}, "Cow cow scowter", 3}} {
		d := model.Document{Entries: []model.Entry{{Text: "Cat cat scatter"}}}
		n, e := Replace(&d, tc.o, false)
		if e != nil || n != tc.n || d.Entries[0].Text != "Cat cat scatter" {
			t.Fatal(n, e, d)
		}
		_, e = Replace(&d, tc.o, true)
		if e != nil || d.Entries[0].Text != tc.want {
			t.Fatal(d, e, tc.want)
		}
	}
	d := model.Document{}
	if _, e := Replace(&d, Operation{Find: "[", Regex: true}, true); e == nil {
		t.Fatal("invalid regex accepted")
	}
}
func TestWrapTransactional(t *testing.T) {
	d := model.Document{Entries: []model.Entry{{Text: "one two three four five six"}, {Text: "thiswordiswaytoolong"}}}
	_, e := Apply(&d, Operation{Kind: "text", Tool: "wrap", MaxChars: 10, MaxLines: 4})
	if e == nil || d.Entries[0].Text != "one two three four five six" {
		t.Fatal("partial edits applied", d, e)
	}
}
