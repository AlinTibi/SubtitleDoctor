package report

import (
	"encoding/csv"
	"encoding/json"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"strings"
	"testing"
)

func TestExports(t *testing.T) {
	r := []model.Report{{Path: "=unsafe.srt", OriginalCount: 3, FinalCount: 2, Issues: []model.Issue{{Code: "overlap", Entry: 2, Message: "Overlap"}}, Repairs: []string{"Renumbered"}, Output: "x.srt"}}
	b, e := Encode(r, "json")
	if e != nil || !json.Valid(b) {
		t.Fatal(string(b), e)
	}
	b, e = Encode(r, "csv")
	if e != nil {
		t.Fatal(e)
	}
	rows, e := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if e != nil || len(rows) != 2 || rows[1][0] != "'=unsafe.srt" {
		t.Fatal(rows, e)
	}
}
