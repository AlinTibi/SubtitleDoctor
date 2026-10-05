package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"strings"
)

func Encode(reports []model.Report, format string) ([]byte, error) {
	if format == "json" {
		return json.MarshalIndent(reports, "", "  ")
	}
	if format != "csv" {
		return nil, fmt.Errorf("unsupported report format")
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if e := w.Write([]string{"source", "original_entries", "final_entries", "issues", "repairs", "output", "error"}); e != nil {
		return nil, e
	}
	for _, r := range reports {
		issues := []string{}
		for _, i := range r.Issues {
			issues = append(issues, fmt.Sprintf("%s[%d]: %s", i.Code, i.Entry, i.Message))
		}
		row := []string{r.Path, fmt.Sprint(r.OriginalCount), fmt.Sprint(r.FinalCount), strings.Join(issues, "; "), strings.Join(r.Repairs, "; "), r.Output, r.Error}
		for i, s := range row {
			if strings.HasPrefix(s, "=") || strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "@") {
				row[i] = "'" + s
			}
		}
		if e := w.Write(row); e != nil {
			return nil, e
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
