package model

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
	SourceHash  string   `json:"-"`
	Path        string   `json:"path"`
	Format      string   `json:"format"`
	Encoding    string   `json:"encoding"`
	LineEnding  string   `json:"lineEnding"`
	BOM         bool     `json:"bom"`
	Entries     []Entry  `json:"entries"`
	Header      []string `json:"header,omitempty"`
	EventFormat []string `json:"eventFormat,omitempty"`
	ParseIssues []Issue  `json:"parseIssues"`
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
