package main

import (
	"context"
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"github.com/AlinTibi/SubtitleDoctor/internal/session"
	"testing"
)

func TestAppPreviewAndEditing(t *testing.T) {
	a := NewApp()
	a.ctx = context.Background()
	d, e := parser.Open("internal/parser/testdata/sample.srt")
	if e != nil {
		t.Fatal(e)
	}
	a.files[d.Path] = session.New(d)
	a.order = []string{d.Path}
	p, e := a.Preview(d.Path, repair.Operation{Kind: "offset", Offset: 1500})
	if e != nil || p.Changed != 3 {
		t.Fatal(p, e)
	}
	v, _ := a.GetFile(d.Path, 0)
	if v.Doc.Entries[0].Start != 3000 {
		t.Fatal("preview mutated session")
	}
	if e = a.Apply(d.Path, repair.Operation{Kind: "repair", Repair: repair.Options{Renumber: true, Duplicates: true}}); e != nil {
		t.Fatal(e)
	}
	v, _ = a.GetFile(d.Path, 0)
	if v.Total != 2 || v.Doc.Entries[0].Index != 1 || !v.CanUndo {
		t.Fatal(v)
	}
	if e = a.History(d.Path, false); e != nil {
		t.Fatal(e)
	}
	v, _ = a.GetFile(d.Path, 0)
	if v.Total != 3 {
		t.Fatal(v)
	}
	v.Doc.Entries[0].Text = "Edited"
	if e = a.EditPage(d.Path, 0, v.Doc.Entries); e != nil {
		t.Fatal(e)
	}
	v, _ = a.GetFile(d.Path, 0)
	if v.Doc.Entries[0].Text != "Edited" {
		t.Fatal(v)
	}
}
