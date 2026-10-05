package repair

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/analyzer"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	timing "github.com/AlinTibi/SubtitleDoctor/internal/sync"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Options struct {
	Renumber    bool `json:"renumber"`
	Empty       bool `json:"empty"`
	Duplicates  bool `json:"duplicates"`
	Order       bool `json:"order"`
	Duration    bool `json:"duration"`
	LineEndings bool `json:"lineEndings"`
	Spacing     bool `json:"spacing"`
	UTF8        bool `json:"utf8"`
	RemoveBOM   bool `json:"removeBOM"`
	Tags        bool `json:"tags"`
}
type Operation struct {
	Kind          string  `json:"kind"`
	Repair        Options `json:"repair"`
	Offset        float64 `json:"offset"`
	From          float64 `json:"from"`
	To            float64 `json:"to"`
	A             float64 `json:"a"`
	B             float64 `json:"b"`
	C             float64 `json:"c"`
	D             float64 `json:"d"`
	Find          string  `json:"find"`
	Replace       string  `json:"replace"`
	Regex         bool    `json:"regex"`
	CaseSensitive bool    `json:"caseSensitive"`
	WholeWord     bool    `json:"wholeWord"`
	Tool          string  `json:"tool"`
	MaxChars      int     `json:"maxChars"`
	MaxLines      int     `json:"maxLines"`
	Format        string  `json:"format"`
}

func Apply(d *model.Document, o Operation) ([]string, error) {
	before := model.Clone(*d)
	log := []string{}
	var err error
	switch o.Kind {
	case "repair":
		log = Fix(d, o.Repair)
	case "offset":
		err = timing.Offset(d, o.Offset)
		log = append(log, fmt.Sprintf("Offset %g ms", o.Offset))
	case "fps":
		err = timing.FPS(d, o.From, o.To)
		log = append(log, fmt.Sprintf("FPS %g → %g", o.From, o.To))
	case "resync":
		err = timing.TwoPoint(d, o.A, o.B, o.C, o.D)
		log = append(log, "Two-point timing correction")
	case "replace":
		var n int
		n, err = Replace(d, o, true)
		log = append(log, fmt.Sprintf("Replaced %d matches", n))
	case "text":
		err = Text(d, o)
		log = append(log, "Text tool: "+o.Tool)
	case "convert":
		if o.Format != "srt" && o.Format != "vtt" && o.Format != "ass" && o.Format != "ssa" {
			err = fmt.Errorf("unsupported target format")
		}
		log = append(log, "Export format: "+o.Format)
	default:
		err = fmt.Errorf("unknown operation %q", o.Kind)
	}
	if err != nil {
		*d = before
		return nil, err
	}
	return log, nil
}
func Fix(d *model.Document, o Options) []string {
	log := []string{}
	entries := []model.Entry{}
	seen := map[string]bool{}
	for i, e := range d.Entries {
		if o.Empty && strings.TrimSpace(analyzer.Plain(e.Text)) == "" {
			log = append(log, fmt.Sprintf("Removed empty entry %d", i+1))
			continue
		}
		key := fmt.Sprintf("%d/%d/%s", e.Start, e.End, e.Text)
		if o.Duplicates && seen[key] {
			log = append(log, fmt.Sprintf("Removed duplicate %d", i+1))
			continue
		}
		seen[key] = true
		if o.Spacing {
			t := normalize(e.Text)
			if t != e.Text {
				e.Text = t
				log = append(log, fmt.Sprintf("Normalized spacing %d", i+1))
			}
		}
		if o.Tags {
			t := CleanHTML(e.Text)
			if t != e.Text {
				e.Text = t
				log = append(log, fmt.Sprintf("Removed unbalanced/unsupported HTML delimiters %d", i+1))
			}
		}
		entries = append(entries, e)
	}
	d.Entries = entries
	if o.Order {
		sort.SliceStable(d.Entries, func(i, j int) bool { return d.Entries[i].Start < d.Entries[j].Start })
		log = append(log, "Sorted entries chronologically")
	}
	if o.Duration {
		for i := range d.Entries {
			e := &d.Entries[i]
			if !e.Invalid && e.Start >= 0 && e.End <= e.Start {
				end := e.Start + 1000
				if i+1 < len(d.Entries) && d.Entries[i+1].Start > e.Start && d.Entries[i+1].Start < end {
					end = d.Entries[i+1].Start
				}
				e.End = end
				log = append(log, fmt.Sprintf("Set entry %d end to %d ms (review timing)", i+1, end))
			}
		}
	}
	if o.Renumber {
		for i := range d.Entries {
			d.Entries[i].Index = i + 1
		}
		log = append(log, "Renumbered entries")
	}
	if o.LineEndings {
		d.LineEnding = "CRLF"
		log = append(log, "Normalized line endings to CRLF")
	}
	if o.UTF8 {
		d.Encoding = "UTF-8"
		log = append(log, "Converted encoding to UTF-8")
	}
	if o.RemoveBOM {
		d.BOM = false
		log = append(log, "Removed BOM")
	}
	issues := []model.Issue{}
	for _, v := range d.ParseIssues {
		if (v.Code == "bom" && o.RemoveBOM) || (v.Code == "encoding" && o.UTF8) || (v.Code == "line_endings" && o.LineEndings) {
			continue
		}
		issues = append(issues, v)
	}
	d.ParseIssues = issues
	return log
}
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	out := []string{}
	blank := false
	for _, l := range lines {
		l = regexp.MustCompile(`[\t ]+`).ReplaceAllString(strings.TrimSpace(l), " ")
		if l == "" && blank {
			continue
		}
		blank = l == ""
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
func expression(o Operation) (*regexp.Regexp, error) {
	if o.Find == "" {
		return nil, fmt.Errorf("find text cannot be empty")
	}
	p := o.Find
	if !o.Regex {
		p = regexp.QuoteMeta(p)
	}
	if o.WholeWord {
		p = `\b(?:` + p + `)\b`
	}
	if !o.CaseSensitive {
		p = "(?i)" + p
	}
	r, e := regexp.Compile(p)
	if e != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", e)
	}
	return r, nil
}
func Replace(d *model.Document, o Operation, apply bool) (int, error) {
	r, e := expression(o)
	if e != nil {
		return 0, e
	}
	n := 0
	for i := range d.Entries {
		n += len(r.FindAllStringIndex(d.Entries[i].Text, -1))
		if apply {
			if o.Regex {
				d.Entries[i].Text = r.ReplaceAllString(d.Entries[i].Text, o.Replace)
			} else {
				d.Entries[i].Text = r.ReplaceAllStringFunc(d.Entries[i].Text, func(string) string { return o.Replace })
			}
		}
	}
	return n, nil
}
func Text(d *model.Document, o Operation) error {
	for i := range d.Entries {
		s := d.Entries[i].Text
		switch o.Tool {
		case "trim":
			lines := strings.Split(s, "\n")
			for j := range lines {
				lines[j] = strings.TrimSpace(lines[j])
			}
			s = strings.Join(lines, "\n")
		case "spaces":
			s = regexp.MustCompile(`[\t ]+`).ReplaceAllString(s, " ")
		case "blank":
			s = regexp.MustCompile(`\n[ \t]*\n(?:[ \t]*\n)+`).ReplaceAllString(s, "\n\n")
		case "upper":
			s = strings.ToUpper(s)
		case "lower":
			s = strings.ToLower(s)
		case "sentence":
			next := true
			s = strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) {
					if next {
						next = false
						return unicode.ToUpper(r)
					}
					return unicode.ToLower(r)
				}
				if r == '.' || r == '!' || r == '?' {
					next = true
				}
				return r
			}, s)
		case "quotes":
			open := true
			s = strings.Map(func(r rune) rune {
				if r == '"' {
					if open {
						open = false
						return '“'
					}
					open = true
					return '”'
				}
				return r
			}, s)
		case "html":
			s = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
		case "ass":
			s = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(s, "")
		case "join":
			s = strings.Join(strings.Fields(s), " ")
		case "wrap":
			if o.MaxChars < 10 || o.MaxLines < 1 {
				return fmt.Errorf("line width must be ≥10 and line count ≥1")
			}
			words := strings.Fields(s)
			lines := []string{}
			line := ""
			for _, w := range words {
				if utf8.RuneCountInString(w) > o.MaxChars {
					return fmt.Errorf("entry %d contains a word longer than the line limit", i+1)
				}
				if line != "" && utf8.RuneCountInString(line+" "+w) > o.MaxChars {
					lines = append(lines, line)
					line = w
				} else {
					if line != "" {
						line += " "
					}
					line += w
				}
			}
			if line != "" {
				lines = append(lines, line)
			}
			if len(lines) > o.MaxLines {
				return fmt.Errorf("entry %d needs %d lines; raise the max lines limit", i+1, len(lines))
			}
			s = strings.Join(lines, "\n")
		default:
			return fmt.Errorf("unknown text tool")
		}
		d.Entries[i].Text = s
	}
	return nil
}
