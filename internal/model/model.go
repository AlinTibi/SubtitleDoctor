package model

import (
	"fmt"
	"strings"
)

// Numbering does not distinguish duplicates, but cue/style metadata does.
func DuplicateKey(d Document, e Entry) string {
	metadata := e.Fields
	if d.Format == "ass" || d.Format == "ssa" {
		metadata = nil
		for i, value := range e.Fields {
			if i < len(d.EventFormat) {
				switch strings.ToLower(strings.TrimSpace(d.EventFormat[i])) {
				case "start", "end", "text":
					continue
				}
			}
			metadata = append(metadata, value)
		}
	}
	return fmt.Sprintf("%d/%d/%q/%q", e.Start, e.End, e.Text, metadata)
}

type Entry struct {
	Index   int      `json:"index"`
	Start   int64    `json:"start"`
	End     int64    `json:"end"`
	Text    string   `json:"text"`
	Fields  []string `json:"fields,omitempty"`
	Invalid bool     `json:"invalid,omitempty"`
}
type Issue struct {
	Code     string `json:"code"`
	Entry    int    `json:"entry"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}
type Document struct {
	SourceHash     string   `json:"-"`
	Path           string   `json:"path"`
	Format         string   `json:"format"`
	Encoding       string   `json:"encoding"`
	OutputEncoding string   `json:"outputEncoding,omitempty"`
	LineEnding     string   `json:"lineEnding"`
	BOM            bool     `json:"bom"`
	Entries        []Entry  `json:"entries"`
	Header         []string `json:"header,omitempty"`
	EventFormat    []string `json:"eventFormat,omitempty"`
	ParseIssues    []Issue  `json:"parseIssues"`
}
type Report struct {
	Path          string   `json:"path"`
	Issues        []Issue  `json:"issues"`
	Repairs       []string `json:"repairs"`
	OriginalCount int      `json:"originalCount"`
	FinalCount    int      `json:"finalCount"`
	Output        string   `json:"output"`
	Error         string   `json:"error,omitempty"`
}

func Clone(d Document) Document {
	d.Entries = append([]Entry(nil), d.Entries...)
	for i := range d.Entries {
		d.Entries[i].Fields = append([]string(nil), d.Entries[i].Fields...)
	}
	d.ParseIssues = append([]Issue(nil), d.ParseIssues...)
	return d
}
