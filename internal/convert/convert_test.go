package convert

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"strings"
	"testing"
)

func TestConversionMatrix(t *testing.T) {
	for _, from := range []string{"srt", "vtt", "ass", "ssa"} {
		d, e := parser.Open("../parser/testdata/sample." + from)
		if e != nil {
			t.Fatal(e)
		}
		for _, to := range []string{"srt", "vtt", "ass", "ssa"} {
			t.Run(from+"_"+to, func(t *testing.T) {
				b, e := Encode(d, to, "UTF-8", "CRLF", false)
				if e != nil {
					t.Fatal(e)
				}
				next, e := parser.Parse("out."+to, b)
				if e != nil {
					t.Fatal(e)
				}
				if len(next.Entries) != len(d.Entries) || next.Entries[0].Start != 3000 || !strings.Contains(next.Entries[0].Text, "Hello, world.") {
					t.Fatal(next)
				}
				if from == to && (from == "ass" || from == "ssa") && !strings.Contains(string(b), "Style: Default") {
					t.Fatal("styles lost")
				}
			})
		}
	}
}
func TestRefuseMalformed(t *testing.T) {
	d, e := parser.Open("../parser/testdata/malformed.srt")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Encode(d, "srt", "UTF-8", "LF", false); e == nil {
		t.Fatal("silently discarded unreadable blocks")
	}
}
func TestUnicodeEncoding(t *testing.T) {
	d, e := parser.Parse("x.srt", []byte("1\n00:00:01,000 --> 00:00:02,000\nȘtefan 日本語"))
	if e != nil {
		t.Fatal(e)
	}
	for _, enc := range []string{"UTF-8", "UTF-16LE"} {
		b, e := Encode(d, "srt", enc, "CRLF", true)
		if e != nil {
			t.Fatal(e)
		}
		p, e := parser.Parse("x.srt", b)
		if e != nil || p.Entries[0].Text != d.Entries[0].Text {
			t.Fatal(p, e)
		}
	}
	if _, e = Encode(d, "srt", "Windows-1252", "LF", false); e == nil {
		t.Fatal("silently lost unsupported characters")
	}
}
