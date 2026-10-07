package convert_test

import (
	"bytes"
	"github.com/AlinTibi/SubtitleDoctor/internal/convert"
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"reflect"
	"testing"
)

func TestEditingRoundTrip(t *testing.T) {
	for _, format := range []string{"srt", "vtt"} {
		for _, eol := range []string{"LF", "CRLF"} {
			t.Run(format+eol, func(t *testing.T) {
				input := "1\n00:00:01,000 --> 00:00:05,000 X1:10 X2:20 Y1:30 Y2:40\n<b><i>Bună țară frumoasă astăzi</i></b>\n"
				if format == "vtt" {
					input = "WEBVTT\n\n00:00:01.000 --> 00:00:05.000 align:start position:10%\n<b><i>Bună țară frumoasă astăzi</i></b>\n"
				}
				d, err := parser.Parse("sample."+format, []byte(input))
				if err != nil {
					t.Fatal(err)
				}
				fields := append([]string(nil), d.Entries[0].Fields...)
				if err := repair.EditCues(&d, []int{0}, "split"); err != nil {
					t.Fatal(err)
				}
				if err := repair.EditCues(&d, []int{0, 1}, "merge"); err != nil {
					t.Fatal(err)
				}
				if _, err := repair.Apply(&d, repair.Operation{Kind: "text", Tool: "wrap", MaxChars: 15, MaxLines: 6}); err != nil {
					t.Fatal(err)
				}
				b, err := convert.Encode(d, format, "UTF-8", eol, false)
				if err != nil {
					t.Fatal(err)
				}
				if eol == "CRLF" && !bytes.Contains(b, []byte("\r\n")) {
					t.Fatal("lost CRLF")
				}
				reopened, err := parser.Parse("saved."+format, b)
				if err != nil {
					t.Fatal(err)
				}
				if reopened.Entries[0].Text != d.Entries[0].Text || reopened.Entries[0].Start != 1000 || reopened.Entries[0].End != 5000 || !reflect.DeepEqual(reopened.Entries[0].Fields, fields) {
					t.Fatal(reopened.Entries, d.Entries)
				}
			})
		}
	}
}
