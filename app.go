package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/analyzer"
	"github.com/AlinTibi/SubtitleDoctor/internal/convert"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"github.com/AlinTibi/SubtitleDoctor/internal/output"
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"github.com/AlinTibi/SubtitleDoctor/internal/report"
	"github.com/AlinTibi/SubtitleDoctor/internal/session"
	"github.com/AlinTibi/SubtitleDoctor/internal/settings"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type App struct {
	ctx    context.Context
	mu     sync.Mutex
	files  map[string]*session.File
	order  []string
	prefs  settings.Settings
	cancel context.CancelFunc
	busy   bool
}
type Summary struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Format   string `json:"format"`
	Encoding string `json:"encoding"`
	Count    int    `json:"count"`
	First    int64  `json:"first"`
	Last     int64  `json:"last"`
	Duration int64  `json:"duration"`
	Issues   int    `json:"issues"`
	Dirty    bool   `json:"dirty"`
	Output   string `json:"output"`
}
type View struct {
	Doc     model.Document `json:"doc"`
	Report  model.Report   `json:"report"`
	Target  string         `json:"target"`
	CanUndo bool           `json:"canUndo"`
	CanRedo bool           `json:"canRedo"`
	Total   int            `json:"total"`
}
type Preview struct {
	Changed  int           `json:"changed"`
	Matches  int           `json:"matches"`
	Examples []model.Entry `json:"examples"`
	Warnings []string      `json:"warnings"`
}
type Progress struct {
	Kind      string `json:"kind"`
	Current   string `json:"current"`
	Done      int    `json:"done"`
	Total     int    `json:"total"`
	Error     string `json:"error"`
	Finished  bool   `json:"finished"`
	Cancelled bool   `json:"cancelled"`
}

func NewApp() *App { return &App{files: map[string]*session.File{}, prefs: settings.Default()} }

// EditCues validates the current UI draft without committing session history.
func (a *App) EditCues(d model.Document, ids []int, action string) ([]model.Entry, error) {
	if err := repair.EditCues(&d, ids, action); err != nil {
		return nil, err
	}
	return d.Entries, nil
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	s, e := settings.Load()
	a.prefs = s
	if e != nil {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.WarningDialog, Title: "Settings could not be loaded", Message: e.Error() + "\nUsing safe defaults."})
	}
}
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	dirty := a.busy
	for _, f := range a.files {
		dirty = dirty || f.Dirty
	}
	a.mu.Unlock()
	if dirty {
		r, e := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog, Title: "Close Subtitle Doctor?", Message: "There are unsaved edits or a running job. Close and discard this session?", Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No"})
		if e != nil || r != "Yes" {
			return true
		}
	}
	a.Cancel()
	return false
}
func (a *App) GetSettings() settings.Settings { a.mu.Lock(); defer a.mu.Unlock(); return a.prefs }
func (a *App) SaveSettings(s settings.Settings) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return fmt.Errorf("wait for the current job")
	}
	if e := settings.Save(s); e != nil {
		return e
	}
	a.prefs = s
	return nil
}
func (a *App) PickFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select output folder"})
}
func (a *App) AddFiles() error {
	paths, e := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: "Add subtitles", Filters: []runtime.FileFilter{{DisplayName: "Subtitle files", Pattern: "*.srt;*.ass;*.ssa;*.vtt"}}})
	if e != nil {
		return e
	}
	if len(paths) == 0 {
		return nil
	}
	return a.ImportPaths(paths)
}
func (a *App) AddFolder() error {
	p, e := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Add folder (recursive)"})
	if e != nil || p == "" {
		return e
	}
	return a.ImportPaths([]string{p})
}
func supported(p string) bool {
	f := strings.ToLower(filepath.Ext(p))
	return f == ".srt" || f == ".ass" || f == ".ssa" || f == ".vtt"
}
func (a *App) begin() (context.Context, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return nil, fmt.Errorf("another job is running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.busy = true
	return ctx, nil
}
func (a *App) finish(p Progress) {
	a.mu.Lock()
	a.busy = false
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.mu.Unlock()
	p.Finished = true
	p.Error = ""
	p.Current = ""
	runtime.EventsEmit(a.ctx, "progress", p)
}
func (a *App) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}
func (a *App) ImportPaths(paths []string) error {
	ctx, e := a.begin()
	if e != nil {
		return e
	}
	go func() {
		p := Progress{Kind: "import"}
		defer func() { a.finish(p) }()
		all := []string{}
		for _, root := range paths {
			if ctx.Err() != nil {
				p.Cancelled = true
				return
			}
			info, err := os.Stat(root)
			if err != nil {
				p.Error = err.Error()
				runtime.EventsEmit(a.ctx, "progress", p)
				continue
			}
			if info.IsDir() {
				err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err != nil {
						runtime.EventsEmit(a.ctx, "progress", Progress{Kind: "import", Current: path, Error: err.Error()})
						return nil
					}
					if !d.IsDir() && supported(path) {
						all = append(all, path)
					}
					return nil
				})
				if err != nil {
					p.Error = err.Error()
				}
			} else {
				all = append(all, root)
			}
		}
		p.Total = len(all)
		for i, path := range all {
			if ctx.Err() != nil {
				p.Cancelled = true
				return
			}
			p.Current = path
			p.Done = i
			p.Error = ""
			abs, err := filepath.Abs(path)
			if err == nil {
				path = filepath.Clean(abs)
			}
			a.mu.Lock()
			exists := a.files[path] != nil
			a.mu.Unlock()
			if !exists {
				d, err := parser.Open(path)
				if err != nil {
					p.Error = err.Error()
				} else {
					a.mu.Lock()
					f := session.New(d)
					f.Report.Issues = analyzer.Scan(d, a.prefs.MaxChars, a.prefs.CPS)
					a.files[path] = f
					a.order = append(a.order, path)
					a.mu.Unlock()
				}
			}
			p.Done = i + 1
			runtime.EventsEmit(a.ctx, "progress", p)
		}
	}()
	return nil
}
func (a *App) Queue() []Summary {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []Summary{}
	for _, path := range a.order {
		f := a.files[path]
		d := f.Doc
		s := Summary{Path: path, Name: filepath.Base(path), Format: d.Format, Encoding: d.Encoding, Count: len(d.Entries), Issues: len(f.Report.Issues), Dirty: f.Dirty, Output: f.Report.Output}
		if len(d.Entries) > 0 {
			s.First = d.Entries[0].Start
			for _, e := range d.Entries {
				if e.Start < s.First {
					s.First = e.Start
				}
				if e.End > s.Last {
					s.Last = e.End
				}
			}
			s.Duration = s.Last - s.First
		}
		out = append(out, s)
	}
	return out
}
func (a *App) GetFile(path string, page int) (View, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.files[path]
	if f == nil {
		return View{}, fmt.Errorf("file is not in the queue")
	}
	d := f.Doc
	start := page * 200
	if start < 0 || start > len(d.Entries) {
		start = 0
	}
	end := start + 200
	if end > len(d.Entries) {
		end = len(d.Entries)
	}
	d.Entries = d.Entries[start:end]
	d = model.Clone(d)
	return View{d, f.Report, f.Target, len(f.Undo) > 0, len(f.Redo) > 0, len(f.Doc.Entries)}, nil
}
func (a *App) Scan(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return fmt.Errorf("wait for current job")
	}
	for p, f := range a.files {
		if path == "" || p == path {
			f.Report.Issues = analyzer.Scan(f.Doc, a.prefs.MaxChars, a.prefs.CPS)
			f.Report.FinalCount = len(f.Doc.Entries)
		}
	}
	return nil
}
func (a *App) Preview(path string, o repair.Operation) (Preview, error) {
	a.mu.Lock()
	f := a.files[path]
	if f == nil {
		a.mu.Unlock()
		return Preview{}, fmt.Errorf("select a file")
	}
	d := model.Clone(f.Doc)
	a.mu.Unlock()
	before := model.Clone(d)
	p := Preview{Examples: []model.Entry{}, Warnings: []string{}}
	if o.Kind == "replace" {
		n, e := repair.Replace(&d, o, false)
		if e != nil {
			return p, e
		}
		p.Matches = n
	}
	if _, e := repair.Apply(&d, o); e != nil {
		return p, e
	}
	for i, v := range d.Entries {
		if i >= len(before.Entries) || v.Text != before.Entries[i].Text || v.Start != before.Entries[i].Start || v.End != before.Entries[i].End || v.Index != before.Entries[i].Index {
			p.Changed++
			if len(p.Examples) < 8 {
				p.Examples = append(p.Examples, v)
			}
		}
	}
	if len(before.Entries) > len(d.Entries) {
		p.Changed += len(before.Entries) - len(d.Entries)
	}
	if o.Kind == "convert" {
		p.Warnings = convert.Warnings(d, o.Format)
	}
	return p, nil
}
func (a *App) Apply(path string, o repair.Operation) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return fmt.Errorf("wait for current job")
	}
	f := a.files[path]
	if f == nil {
		return fmt.Errorf("select a file")
	}
	if e := f.Apply(o); e != nil {
		return e
	}
	f.Report.Issues = analyzer.Scan(f.Doc, a.prefs.MaxChars, a.prefs.CPS)
	f.Report.FinalCount = len(f.Doc.Entries)
	return nil
}
func (a *App) History(path string, redo bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return fmt.Errorf("wait for current job")
	}
	f := a.files[path]
	if f == nil {
		return fmt.Errorf("select a file")
	}
	if e := f.History(redo); e != nil {
		return e
	}
	f.Report.Issues = analyzer.Scan(f.Doc, a.prefs.MaxChars, a.prefs.CPS)
	f.Report.FinalCount = len(f.Doc.Entries)
	return nil
}
func (a *App) EditPage(path string, page int, entries []model.Entry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.busy {
		return fmt.Errorf("wait for current job")
	}
	f := a.files[path]
	if f == nil {
		return fmt.Errorf("select a file")
	}
	start := page * 200
	end := start + 200
	if start < 0 || start > len(f.Doc.Entries) {
		return fmt.Errorf("invalid page")
	}
	if end > len(f.Doc.Entries) {
		end = len(f.Doc.Entries)
	}
	all := append([]model.Entry{}, f.Doc.Entries[:start]...)
	all = append(all, entries...)
	all = append(all, f.Doc.Entries[end:]...)
	if e := f.Edit(all); e != nil {
		return e
	}
	f.Report.Issues = analyzer.Scan(f.Doc, a.prefs.MaxChars, a.prefs.CPS)
	f.Report.FinalCount = len(f.Doc.Entries)
	return nil
}
func (a *App) StartBatch(paths []string, o repair.Operation, save bool, encoding string) error {
	a.mu.Lock()
	prefs := a.prefs
	a.mu.Unlock()
	confirmed := false
	if save && prefs.Overwrite {
		r, e := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.WarningDialog, Title: "Replace original subtitle files?", Message: "This will replace each selected source file. A uniquely named .bak backup is required and will be created first. Review your edits and output format before confirming.", Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No"})
		if e != nil {
			return e
		}
		if r != "Yes" {
			return fmt.Errorf("replacement cancelled")
		}
		confirmed = true
	}
	ctx, e := a.begin()
	if e != nil {
		return e
	}
	go func() {
		p := Progress{Kind: "batch", Total: len(paths)}
		defer func() { a.finish(p) }()
		for i, path := range paths {
			if ctx.Err() != nil {
				p.Cancelled = true
				return
			}
			p.Current = path
			p.Done = i
			p.Error = ""
			a.mu.Lock()
			f := a.files[path]
			var d model.Document
			target := ""
			if f != nil {
				d = model.Clone(f.Doc)
				target = f.Target
			}
			a.mu.Unlock()
			if f == nil {
				p.Error = "File no longer queued"
			} else {
				logs := []string{}
				var err error
				if o.Kind != "" {
					logs, err = repair.Apply(&d, o)
					if o.Kind == "convert" {
						target = o.Format
					}
				}
				saved := ""
				if err == nil && save {
					d, saved, err = writeDocument(ctx, d, target, encoding, prefs, confirmed)
				}
				a.mu.Lock()
				if err == nil && (ctx.Err() == nil || saved != "") {
					if o.Kind != "" || (saved != "" && prefs.Overwrite) {
						f.Commit(d, target, logs)
					}
					if saved != "" {
						f.Doc.SourceHash = d.SourceHash
						f.Report.Output = saved
						f.Dirty = false
					}
					f.Report.Issues = analyzer.Scan(f.Doc, prefs.MaxChars, prefs.CPS)
					f.Report.FinalCount = len(f.Doc.Entries)
					f.Report.Error = ""
				} else if err != nil {
					f.Report.Error = err.Error()
				}
				a.mu.Unlock()
				if err != nil {
					p.Error = err.Error()
				}
			}
			p.Done = i + 1
			runtime.EventsEmit(a.ctx, "progress", p)
			if ctx.Err() != nil {
				p.Cancelled = true
				return
			}
		}
	}()
	return nil
}

// An empty explicitEncoding means an operation's encoding intent takes priority
// over settings. Only Save / Export supplies a nonempty explicit override.
func writeDocument(ctx context.Context, d model.Document, target, explicitEncoding string, prefs settings.Settings, confirmed bool) (model.Document, string, error) {
	encoding := explicitEncoding
	if encoding == "" {
		encoding = d.OutputEncoding
	}
	if encoding == "" {
		encoding = prefs.Encoding
	}
	data, err := convert.Encode(d, target, encoding, d.LineEnding, d.BOM)
	if err != nil {
		return d, "", err
	}
	written := d
	if prefs.Overwrite {
		current, err := os.ReadFile(d.Path)
		if err != nil {
			return d, "", err
		}
		if fmt.Sprintf("%x", sha256.Sum256(current)) != d.SourceHash {
			return d, "", fmt.Errorf("source changed since import; reopen the file before replacing it")
		}
		written, err = parser.ParseEncoded(d.Path, data, encoding)
		if err != nil {
			return d, "", fmt.Errorf("cannot reparse replacement: %w", err)
		}
		written.OutputEncoding = encoding
	}
	if err := ctx.Err(); err != nil {
		return d, "", err
	}
	saved, err := output.Save(d.Path, prefs.OutputFolder, prefs.Suffix, target, data, prefs.Overwrite, confirmed)
	if err != nil {
		return d, "", err
	}
	return written, saved, nil
}
func (a *App) ExportReport(format string) (string, error) {
	a.mu.Lock()
	reports := []model.Report{}
	for _, path := range a.order {
		reports = append(reports, a.files[path].Report)
	}
	a.mu.Unlock()
	data, e := report.Encode(reports, format)
	if e != nil {
		return "", e
	}
	p, e := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Export processing report", DefaultFilename: "subtitle-report." + format})
	if e != nil || p == "" {
		return "", e
	}
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return "", fmt.Errorf("choose a new report filename: %w", e)
	}
	_, e = f.Write(data)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	return p, e
}
