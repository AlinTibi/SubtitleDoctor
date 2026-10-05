package convert

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/markup"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"regexp"
	"strings"
)

func Warnings(d model.Document, target string) []string {
	out := []string{}
	if d.Format != target {
		if d.Format == "ass" || d.Format == "ssa" {
			out = append(out, "ASS/SSA style definitions, positioning, karaoke and drawing features may be lost. Basic bold, italic and underline are converted.")
		}
		if d.Format == "vtt" {
			out = append(out, "WebVTT cue identifiers, positioning, regions and CSS will be lost.")
		}
		if target == "ass" || target == "ssa" {
			out = append(out, "ASS/SSA times have centisecond precision; milliseconds are rounded down.")
		}
	}
	return out
}
func text(s, from, to string) string {
	ass := func(f string) bool { return f == "ass" || f == "ssa" }
	if ass(from) && !ass(to) {
		for _, t := range []string{"b", "i", "u"} {
			s = strings.ReplaceAll(s, `{\`+t+`1}`, "<"+t+">")
			s = strings.ReplaceAll(s, `{\`+t+`0}`, "</"+t+">")
		}
		s = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(s, "")
	}
	if !ass(from) && ass(to) {
		for _, t := range []string{"b", "i", "u"} {
			s = strings.ReplaceAll(s, "<"+t+">", `{\`+t+`1}`)
			s = strings.ReplaceAll(s, "</"+t+">", `{\`+t+`0}`)
		}
		s = markup.StripHTML(s)
	}
	if ass(to) {
		s = strings.ReplaceAll(s, "\n", `\N`)
	}
	return s
}
func Encode(d model.Document, target, encoding, lineEnding string, bom bool) ([]byte, error) {
	if target != "srt" && target != "vtt" && target != "ass" && target != "ssa" {
		return nil, fmt.Errorf("unsupported output format")
	}
	for _, v := range d.ParseIssues {
		if v.Code == "malformed" {
			return nil, fmt.Errorf("unreadable blocks remain; exporting would lose data. Correct the source file first")
		}
	}
	for i, e := range d.Entries {
		if e.Invalid || e.Start < 0 || e.End <= e.Start {
			return nil, fmt.Errorf("entry %d has invalid timing; correct it before export", i+1)
		}
	}
	var b strings.Builder
	if target == "vtt" {
		if d.Format == target && len(d.Header) > 0 {
			b.WriteString(strings.Join(d.Header, "\n"))
			b.WriteString("\n\n")
		} else {
			b.WriteString("WEBVTT\n\n")
		}
	}
	ass := target == "ass" || target == "ssa"
	format := []string{"Layer", "Start", "End", "Style", "Name", "MarginL", "MarginR", "MarginV", "Effect", "Text"}
	var trailer []string
	if ass {
		if d.Format == target && len(d.Header) > 0 {
			at := -1
			for i, line := range d.Header {
				if strings.EqualFold(strings.TrimSpace(line), "[Events]") {
					at = i
					break
				}
			}
			if at < 0 {
				return nil, fmt.Errorf("ASS Events section missing")
			}
			b.WriteString(strings.Join(d.Header[:at+1], "\n") + "\n")
			trailer = d.Header[at+1:]
			format = d.EventFormat
		} else if target == "ssa" {
			b.WriteString("[Script Info]\nScriptType: v4.00\n\n[V4 Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, TertiaryColour, BackColour, Bold, Italic, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, AlphaLevel, Encoding\nStyle: Default,Arial,24,16777215,16777215,0,0,0,0,1,2,1,2,10,10,10,0,1\n\n[Events]\n")
			format[0] = "Marked"
		} else {
			b.WriteString("[Script Info]\nScriptType: v4.00+\nPlayResX: 1920\nPlayResY: 1080\n\n[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\nStyle: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,1,2,20,20,20,1\n\n[Events]\n")
		}
		if len(format) == 0 {
			return nil, fmt.Errorf("ASS event format missing")
		}
		b.WriteString("Format: " + strings.Join(format, ", ") + "\n")
	}
	for i, e := range d.Entries {
		if ass {
			fields := []string{"0", "", "", "Default", "", "0", "0", "0", "", ""}
			if target == "ssa" {
				fields[0] = "Marked=0"
			}
			if d.Format == target {
				fields = append([]string(nil), e.Fields...)
			}
			if len(fields) != len(format) {
				return nil, fmt.Errorf("entry %d has mismatched ASS fields", i+1)
			}
			for j, k := range format {
				switch strings.ToLower(k) {
				case "start":
					fields[j] = parser.FormatTime(e.Start, target)
				case "end":
					fields[j] = parser.FormatTime(e.End, target)
				case "text":
					fields[j] = text(e.Text, d.Format, target)
				}
			}
			b.WriteString("Dialogue: " + strings.Join(fields, ",") + "\n")
		} else {
			if target == "srt" {
				fmt.Fprintf(&b, "%d\n", i+1)
			} else if d.Format == target && len(e.Fields) > 0 {
				b.WriteString(e.Fields[0] + "\n")
			}
			fmt.Fprintf(&b, "%s --> %s", parser.FormatTime(e.Start, target), parser.FormatTime(e.End, target))
			if target == "vtt" && d.Format == target && len(e.Fields) > 1 {
				b.WriteString(" " + e.Fields[1])
			}
			b.WriteString("\n" + text(e.Text, d.Format, target) + "\n\n")
		}
	}
	if ass && len(trailer) > 0 {
		b.WriteString(strings.Join(trailer, "\n"))
	}
	s := b.String()
	if lineEnding == "CRLF" {
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	switch encoding {
	case "UTF-8":
		if bom {
			s = "\ufeff" + s
		}
		return []byte(s), nil
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes([]byte(s))
	case "Windows-1252":
		return charmap.Windows1252.NewEncoder().Bytes([]byte(s))
	default:
		return nil, fmt.Errorf("unsupported output encoding %q", encoding)
	}
}
