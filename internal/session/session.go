package session

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"reflect"
)

type Snapshot struct {
	Doc     model.Document
	Repairs []string
	Target  string
}
type File struct {
	Doc      model.Document
	Original model.Document
	Report   model.Report
	Target   string
	Undo     []Snapshot
	Redo     []Snapshot
	Dirty    bool
}

func New(d model.Document) *File {
	return &File{Doc: d, Original: model.Clone(d), Target: d.Format, Report: model.Report{Path: d.Path, OriginalCount: len(d.Entries), FinalCount: len(d.Entries), Issues: []model.Issue{}, Repairs: []string{}}}
}
func (f *File) snapshot() Snapshot {
	return Snapshot{model.Clone(f.Doc), append([]string{}, f.Report.Repairs...), f.Target}
}
func (f *File) push() {
	f.Undo = append(f.Undo, f.snapshot())
	bytes := 0
	for _, e := range f.Doc.Entries {
		bytes += 96 + len(e.Text)
		for _, field := range e.Fields {
			bytes += len(field) + 16
		}
	}
	limit := 100
	if bytes > 0 && 64*1024*1024/bytes < limit {
		limit = 64 * 1024 * 1024 / bytes
		if limit < 1 {
			limit = 1
		}
	}
	if len(f.Undo) > limit {
		f.Undo = append([]Snapshot(nil), f.Undo[len(f.Undo)-limit:]...)
	}
	f.Redo = nil
}
func (f *File) Apply(o repair.Operation) error {
	d := model.Clone(f.Doc)
	logs, e := repair.Apply(&d, o)
	if e != nil {
		return e
	}
	target := f.Target
	if o.Kind == "convert" {
		target = o.Format
	}
	f.Commit(d, target, logs)
	return nil
}
func (f *File) Commit(d model.Document, target string, logs []string) {
	f.push()
	f.Doc = d
	f.Target = target
	f.Report.Repairs = append(f.Report.Repairs, logs...)
	f.Dirty = true
}
func (f *File) History(redo bool) error {
	stack := &f.Undo
	other := &f.Redo
	if redo {
		stack = &f.Redo
		other = &f.Undo
	}
	if len(*stack) == 0 {
		return fmt.Errorf("no history available")
	}
	*other = append(*other, f.snapshot())
	s := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	f.Doc = s.Doc
	f.Report.Repairs = s.Repairs
	f.Target = s.Target
	f.Dirty = !reflect.DeepEqual(f.Doc, f.Original) || f.Target != f.Original.Format
	return nil
}
func (f *File) Edit(entries []model.Entry) error {
	if len(entries) > 1000000 {
		return fmt.Errorf("too many entries")
	}
	for i, e := range entries {
		if !e.Invalid && (e.Start < 0 || e.End <= e.Start) {
			return fmt.Errorf("entry %d needs nonnegative start and end later than start", i+1)
		}
	}
	f.push()
	f.Doc.Entries = entries
	for i := range f.Doc.Entries {
		f.Doc.Entries[i].Index = i + 1
	}
	issues := []model.Issue{}
	invalid := false
	for _, e := range f.Doc.Entries {
		invalid = invalid || e.Invalid
	}
	for _, v := range f.Doc.ParseIssues {
		if v.Code != "timestamp" || invalid {
			issues = append(issues, v)
		}
	}
	f.Doc.ParseIssues = issues
	f.Report.Repairs = append(f.Report.Repairs, "Manual entry edit")
	f.Dirty = true
	return nil
}
