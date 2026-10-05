package parser

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var stamp = regexp.MustCompile(`^(\d{1,6}):([0-5]\d):([0-5]\d)[,.](\d{2,3})$`)

func Time(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if strings.Count(s, ":") == 1 {
		s = "00:" + s
	}
	m := stamp.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid timestamp %q", s)
	}
	h, _ := strconv.ParseInt(m[1], 10, 64)
	n, _ := strconv.ParseInt(m[2], 10, 64)
	sec, _ := strconv.ParseInt(m[3], 10, 64)
	ms, _ := strconv.ParseInt(m[4], 10, 64)
	if len(m[4]) == 2 {
		ms *= 10
	}
	return ((h*60+n)*60+sec)*1000 + ms, nil
}
func FormatTime(t int64, format string) string {
	if t < 0 {
		t = 0
	}
	h := t / 3600000
	n := t / 60000 % 60
	s := t / 1000 % 60
	ms := t % 1000
	if format == "ass" || format == "ssa" {
		return fmt.Sprintf("%d:%02d:%02d.%02d", h, n, s, ms/10)
	}
	sep := ","
	if format == "vtt" {
		sep = "."
	}
	return fmt.Sprintf("%02d:%02d:%02d%s%03d", h, n, s, sep, ms)
}
func Open(path string) (model.Document, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return model.Document{}, e
	}
	return Parse(path, b)
}

// ParseEncoded uses a known output encoding instead of heuristic import
// detection. Some Windows-1252 byte sequences are also valid UTF-8.
func ParseEncoded(path string, data []byte, encoding string) (model.Document, error) {
	decoded := data
	if encoding == "Windows-1252" {
		var err error
		decoded, err = charmap.Windows1252.NewDecoder().Bytes(data)
		if err != nil {
			return model.Document{}, err
		}
	} else if encoding != "UTF-8" && encoding != "UTF-16LE" {
		return model.Document{}, fmt.Errorf("unsupported known encoding %q", encoding)
	}
	d, err := Parse(path, decoded)
	if err != nil {
		return d, err
	}
	d.SourceHash = fmt.Sprintf("%x", sha256.Sum256(data))
	d.Encoding = encoding
	d.BOM = encoding == "UTF-16LE" || (encoding == "UTF-8" && bytes.HasPrefix(data, []byte{239, 187, 191}))
	return d, nil
}
func Parse(path string, b []byte) (model.Document, error) {
	f := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if f != "srt" && f != "vtt" && f != "ass" && f != "ssa" {
		return model.Document{}, fmt.Errorf("unsupported format: %s", f)
	}
	d := model.Document{Path: path, Format: f, Encoding: "UTF-8", LineEnding: "LF", Entries: []model.Entry{}, ParseIssues: []model.Issue{}}
	d.SourceHash = fmt.Sprintf("%x", sha256.Sum256(b))
	issue := func(code, msg string) {
		d.ParseIssues = append(d.ParseIssues, model.Issue{Code: code, Message: msg, Severity: "warning"})
	}
	if bytes.HasPrefix(b, []byte{239, 187, 191}) {
		d.BOM = true
		b = b[3:]
		issue("bom", "UTF-8 BOM present")
	}
	var err error
	if bytes.HasPrefix(b, []byte{255, 254}) || bytes.HasPrefix(b, []byte{254, 255}) {
		d.BOM = true
		d.Encoding = "UTF-16LE"
		order := unicode.LittleEndian
		if b[0] == 254 {
			order = unicode.BigEndian
			d.Encoding = "UTF-16BE"
		}
		b, err = unicode.UTF16(order, unicode.ExpectBOM).NewDecoder().Bytes(b)
	} else if !utf8.Valid(b) {
		b, err = charmap.Windows1252.NewDecoder().Bytes(b)
		d.Encoding = "Windows-1252 (assumed)"
		issue("encoding", "Invalid UTF-8; decoded as Windows-1252. Verify text before saving.")
	}
	if err != nil {
		return d, fmt.Errorf("decode: %w", err)
	}
	s := string(b)
	crlf := strings.Count(s, "\r\n")
	lf := strings.Count(s, "\n") - crlf
	cr := strings.Count(s, "\r") - crlf
	if crlf > 0 {
		d.LineEnding = "CRLF"
	}
	if (crlf > 0 && lf > 0) || cr > 0 {
		issue("line_endings", "Inconsistent or legacy line endings")
	}
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	if f == "ass" || f == "ssa" {
		parseASS(&d, s)
	} else {
		parseBlocks(&d, s)
	}
	if len(d.Entries) == 0 {
		return d, fmt.Errorf("no readable subtitle entries; %d malformed blocks", len(d.ParseIssues))
	}
	return d, nil
}
func bad(d *model.Document, n int, msg string) {
	d.ParseIssues = append(d.ParseIssues, model.Issue{Code: "malformed", Entry: n, Message: msg, Severity: "error"})
}
func parseBlocks(d *model.Document, s string) {
	blocks := regexp.MustCompile(`\n[ \t]*\n`).Split(strings.Trim(s, "\n"), -1)
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		if strings.HasPrefix(lines[0], "WEBVTT") {
			d.Header = append(d.Header, lines...)
			continue
		}
		if d.Format == "vtt" && (strings.HasPrefix(lines[0], "NOTE") || lines[0] == "STYLE" || lines[0] == "REGION") {
			d.Header = append(d.Header, "", block)
			continue
		}
		pos := -1
		for i, l := range lines {
			if strings.Contains(l, "-->") {
				pos = i
				break
			}
		}
		if pos < 0 || pos > 1 {
			bad(d, len(d.Entries)+1, "Unrecognized block: "+lines[0])
			continue
		}
		e := model.Entry{Index: len(d.Entries) + 1}
		if d.Format == "vtt" {
			e.Fields = []string{"", ""}
		}
		if pos == 1 {
			if d.Format == "srt" {
				e.Index, _ = strconv.Atoi(strings.TrimSpace(lines[0]))
			} else {
				e.Fields[0] = lines[0]
			}
		} else if d.Format == "srt" {
			e.Index = 0
		}
		pair := strings.SplitN(lines[pos], "-->", 2)
		right := strings.Fields(pair[1])
		a, ae := Time(pair[0])
		var z int64
		var ze error
		if len(right) == 0 {
			ze = fmt.Errorf("missing end time")
		} else {
			z, ze = Time(right[0])
			if len(right) > 1 {
				if d.Format == "vtt" {
					e.Fields[1] = strings.Join(right[1:], " ")
				} else {
					e.Fields = append(e.Fields, strings.Join(right[1:], " "))
				}
			}
		}
		e.Start = a
		e.End = z
		e.Invalid = ae != nil || ze != nil
		if e.Invalid {
			d.ParseIssues = append(d.ParseIssues, model.Issue{Code: "timestamp", Entry: len(d.Entries) + 1, Message: "Invalid timestamp: " + lines[pos], Severity: "error"})
		}
		e.Text = strings.Join(lines[pos+1:], "\n")
		d.Entries = append(d.Entries, e)
	}
}
func parseASS(d *model.Document, s string) {
	inEvents := false
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			inEvents = strings.EqualFold(trim, "[Events]")
		}
		if inEvents && strings.HasPrefix(trim, "Format:") {
			if len(d.Entries) > 0 {
				bad(d, len(d.Entries)+1, "Multiple ASS event formats are unsupported; correct the source before export")
			}
			d.EventFormat = strings.Split(strings.TrimSpace(strings.TrimPrefix(trim, "Format:")), ",")
			for i := range d.EventFormat {
				d.EventFormat[i] = strings.TrimSpace(d.EventFormat[i])
			}
			continue
		}
		if inEvents && strings.HasPrefix(trim, "Dialogue:") {
			if len(d.EventFormat) == 0 {
				d.EventFormat = []string{"Layer", "Start", "End", "Style", "Name", "MarginL", "MarginR", "MarginV", "Effect", "Text"}
				if d.Format == "ssa" {
					d.EventFormat[0] = "Marked"
				}
			}
			hasStart, hasEnd, hasText := false, false, false
			for _, k := range d.EventFormat {
				switch strings.ToLower(k) {
				case "start":
					hasStart = true
				case "end":
					hasEnd = true
				case "text":
					hasText = true
				}
			}
			if !hasStart || !hasEnd || !hasText || !strings.EqualFold(d.EventFormat[len(d.EventFormat)-1], "Text") {
				bad(d, len(d.Entries)+1, "ASS Format requires Start, End and Text as its final field")
				continue
			}
			event := strings.TrimLeft(line, " \t")
			fields := strings.SplitN(strings.TrimLeft(strings.TrimPrefix(event, "Dialogue:"), " \t"), ",", len(d.EventFormat))
			if len(fields) != len(d.EventFormat) {
				bad(d, len(d.Entries)+1, "Malformed Dialogue event")
				continue
			}
			e := model.Entry{Index: len(d.Entries) + 1, Fields: fields}
			for i, k := range d.EventFormat {
				switch strings.ToLower(k) {
				case "start":
					v, err := Time(fields[i])
					e.Start = v
					e.Invalid = e.Invalid || err != nil
				case "end":
					v, err := Time(fields[i])
					e.End = v
					e.Invalid = e.Invalid || err != nil
				case "text":
					e.Text = strings.ReplaceAll(strings.ReplaceAll(fields[i], `\N`, "\n"), `\n`, "\n")
				}
			}
			if e.Invalid {
				d.ParseIssues = append(d.ParseIssues, model.Issue{Code: "timestamp", Entry: e.Index, Message: "Invalid Dialogue timestamp", Severity: "error"})
			}
			d.Entries = append(d.Entries, e)
			continue
		}
		d.Header = append(d.Header, line)
	}
}
