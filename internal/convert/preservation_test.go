package convert

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"strings"
	"testing"
)

func TestVTTSettingsWithoutIdentifier(t *testing.T) {
	d, e := parser.Parse("x.vtt", []byte("WEBVTT\n\n00:00:03.000 --> 00:00:05.000 align:start\nHello"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := Encode(d, "vtt", "UTF-8", "LF", false)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(b), "00:00:05.000 align:start") {
		t.Fatal(string(b))
	}
}
func TestASSTrailingSections(t *testing.T) {
	d, e := parser.Open("../parser/testdata/sample.ass")
	if e != nil {
		t.Fatal(e)
	}
	d.Header = append(d.Header, "[Fonts]", "fontname: sample")
	b, e := Encode(d, "ass", "UTF-8", "LF", false)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Index(string(b), "Dialogue:") > strings.Index(string(b), "[Fonts]") {
		t.Fatal("dialogue placed in Fonts section")
	}
	again, e := parser.Parse("x.ass", b)
	if e != nil || len(again.Entries) != 1 {
		t.Fatal(again, e)
	}
}
