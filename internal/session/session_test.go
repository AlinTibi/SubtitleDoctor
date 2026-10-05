package session

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"testing"
)

func TestUndoRedo(t *testing.T) {
	f := New(model.Document{Format: "srt", Entries: []model.Entry{{Index: 7, Start: 3000, End: 5000, Text: "Hello"}}})
	if e := f.Apply(repair.Operation{Kind: "offset", Offset: 1500}); e != nil {
		t.Fatal(e)
	}
	if e := f.History(false); e != nil || f.Doc.Entries[0].Start != 3000 {
		t.Fatal(f, e)
	}
	if e := f.History(true); e != nil || f.Doc.Entries[0].Start != 4500 {
		t.Fatal(f, e)
	}
	if e := f.Apply(repair.Operation{Kind: "offset", Offset: -10000}); e == nil || len(f.Undo) != 1 {
		t.Fatal("failed operation affected history")
	}
}
