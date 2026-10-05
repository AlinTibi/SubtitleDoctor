package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/convert"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"github.com/AlinTibi/SubtitleDoctor/internal/parser"
	"github.com/AlinTibi/SubtitleDoctor/internal/repair"
	"github.com/AlinTibi/SubtitleDoctor/internal/settings"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

func TestRepairSaveEncodingPrecedence(t *testing.T) {
	prefs := settings.Default()
	prefs.Encoding = "Windows-1252"
	prefs.OutputFolder = t.TempDir()
	original := append([]byte("1\n00:00:01,000 --> 00:00:02,000\nCaf"), 0xe9, '\n')
	src := filepath.Join(t.TempDir(), "film.srt")
	if err := os.WriteFile(src, original, 0600); err != nil {
		t.Fatal(err)
	}
	d, err := parser.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	repair.Fix(&d, repair.Options{UTF8: true, RemoveBOM: true})
	for _, tc := range []struct {
		explicit  string
		validUTF8 bool
	}{{"", true}, {"Windows-1252", false}, {"UTF-8", true}} {
		_, saved, err := writeDocument(context.Background(), d, "srt", tc.explicit, prefs, false)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(saved)
		if err != nil {
			t.Fatal(err)
		}
		if utf8.Valid(b) != tc.validUTF8 {
			t.Fatalf("override %q wrote incorrect encoding: %x", tc.explicit, b)
		}
		if tc.validUTF8 && !bytes.Contains(b, []byte("Café")) {
			t.Fatal("dialogue corrupted")
		}
	}
	untouched, _ := os.ReadFile(src)
	if !bytes.Equal(original, untouched) {
		t.Fatal("source modified")
	}
	// Settings remain the fallback until a repair chooses an output encoding.
	imported, err := parser.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	_, saved, err := writeDocument(context.Background(), imported, "srt", "", prefs, false)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(saved)
	if utf8.Valid(b) {
		t.Fatal("preferred legacy fallback ignored")
	}
}

func TestReplacementEncodingAndBOMState(t *testing.T) {
	for _, tc := range []struct {
		input, explicit string
		repairUTF8      bool
		want            string
		bom             bool
	}{
		{"UTF-16LE", "", true, "UTF-8", false},
		{"UTF-8", "UTF-16LE", false, "UTF-16LE", true},
		{"UTF-16LE", "Windows-1252", true, "Windows-1252", false},
		{"UTF-8", "UTF-8", false, "UTF-8", true},
	} {
		t.Run(tc.input+tc.explicit, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "replace.srt")
			seed := model.Document{Format: "srt", Entries: []model.Entry{{Index: 1, Start: 1000, End: 2000, Text: "Café"}}}
			original, err := convert.Encode(seed, "srt", tc.input, "LF", true)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(src, original, 0600); err != nil {
				t.Fatal(err)
			}
			d, err := parser.Open(src)
			if err != nil {
				t.Fatal(err)
			}
			if tc.repairUTF8 {
				repair.Fix(&d, repair.Options{UTF8: true, RemoveBOM: true})
			}
			prefs := settings.Default()
			prefs.Encoding = "Windows-1252"
			prefs.Overwrite = true
			written, saved, err := writeDocument(context.Background(), d, "srt", tc.explicit, prefs, true)
			if err != nil {
				t.Fatal(err)
			}
			if saved != src || written.Encoding != tc.want || written.BOM != tc.bom || written.OutputEncoding != tc.want {
				t.Fatalf("incorrect replacement state: %+v", written)
			}
			actual, _ := os.ReadFile(src)
			if written.SourceHash != fmt.Sprintf("%x", sha256.Sum256(actual)) {
				t.Fatal("stale source hash")
			}
			backup, _ := os.ReadFile(src + ".bak")
			if !bytes.Equal(backup, original) {
				t.Fatal("backup mismatch")
			}
			parsed, err := parser.Parse(src, actual)
			if err != nil || parsed.Entries[0].Text != "Café" || parsed.BOM != written.BOM {
				t.Fatal("replacement does not round trip", parsed, err)
			}
			// A subsequent automatic save uses the newly written encoding and BOM.
			next, _, err := writeDocument(context.Background(), written, "srt", "", prefs, true)
			if err != nil || next.Encoding != written.Encoding || next.BOM != written.BOM {
				t.Fatal(next, err)
			}
		})
	}
}

func TestReplacementKnownLegacyEncoding(t *testing.T) {
	src := filepath.Join(t.TempDir(), "ambiguous.srt")
	seed := model.Document{Format: "srt", Entries: []model.Entry{{Index: 1, Start: 1000, End: 2000, Text: "Ã©"}}}
	original, err := convert.Encode(seed, "srt", "UTF-8", "LF", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(src, original, 0600); err != nil {
		t.Fatal(err)
	}
	d, err := parser.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	prefs := settings.Default()
	prefs.Overwrite = true
	written, _, err := writeDocument(context.Background(), d, "srt", "Windows-1252", prefs, true)
	if err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(src)
	if !utf8.Valid(actual) {
		t.Fatal("test requires ambiguous UTF-8-valid legacy bytes")
	}
	if written.Entries[0].Text != "Ã©" || written.Encoding != "Windows-1252" || written.BOM {
		t.Fatal("known legacy encoding was misdetected", written)
	}
	if written.SourceHash != fmt.Sprintf("%x", sha256.Sum256(actual)) {
		t.Fatal("wrong source hash")
	}
}
