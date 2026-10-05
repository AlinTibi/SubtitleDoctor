package parser

import (
	"os"
	"strings"
	"testing"
)

func TestFormats(t *testing.T) {
	for _, f := range []string{"srt", "vtt", "ass", "ssa"} {
		t.Run(f, func(t *testing.T) {
			d, e := Open("testdata/sample." + f)
			if e != nil {
				t.Fatal(e)
			}
			if len(d.Entries) == 0 || d.Entries[0].Start != 3000 || d.Entries[0].End != 5000 || !strings.Contains(d.Entries[0].Text, "Hello, world.") {
				t.Fatalf("unexpected document: %+v", d)
			}
			if f == "srt" && d.Entries[0].Index != 7 {
				t.Fatal("parser silently renumbered")
			}
		})
	}
}
func TestMalformed(t *testing.T) {
	b, e := os.ReadFile("testdata/malformed.srt")
	if e != nil {
		t.Fatal(e)
	}
	d, e := Parse("bad.srt", b)
	if e != nil {
		t.Fatal(e)
	}
	if !d.Entries[0].Invalid || len(d.ParseIssues) != 2 {
		t.Fatalf("missing diagnostics: %+v", d)
	}
}
func TestEncoding(t *testing.T) {
	for _, s := range []string{"Hello Ștefan 日本語", "\ufeffHello Ștefan"} {
		d, e := Parse("x.srt", []byte("1\n00:00:00,000 --> 00:00:02,000\n"+s))
		if e != nil || d.Entries[0].Text != s {
			t.Fatalf("unicode changed: %+v %v", d, e)
		}
	}
	d, e := Parse("x.srt", []byte("\ufeff1\r\n00:00:00,000 --> 00:00:02,000\r\nTest\n"))
	if e != nil || !d.BOM || d.LineEnding != "CRLF" || len(d.ParseIssues) < 2 {
		t.Fatalf("missing BOM/line-ending diagnostics: %+v %v", d, e)
	}
	d, e = Parse("x.srt", append([]byte("1\n00:00:00,000 --> 00:00:02,000\n"), 0xe9))
	if e != nil || d.Encoding != "Windows-1252 (assumed)" || d.Entries[0].Text != "é" {
		t.Fatal(d, e)
	}
}
func TestTimestampValidation(t *testing.T) {
	for _, v := range []string{"00:60:00,000", "-1:00:00.000", "00:00:00", "nonsense"} {
		if _, e := Time(v); e == nil {
			t.Errorf("accepted %s", v)
		}
	}
	v, e := Time("1:02:03.45")
	if e != nil || v != 3723450 {
		t.Fatal(v, e)
	}
}
func TestScanPreservesSpacing(t *testing.T) {
	d, e := Parse("x.srt", []byte("1\n00:00:03,000 --> 00:00:05,000\n  Hello  \n"))
	if e != nil || d.Entries[0].Text != "  Hello  " {
		t.Fatal(d, e)
	}
}
