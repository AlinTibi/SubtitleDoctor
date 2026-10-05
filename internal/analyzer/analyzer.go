package analyzer

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var tags = regexp.MustCompile(`<[^>]*>|\{[^}]*\}`)
var htmlTags = regexp.MustCompile(`<[^>]+>`)

func Plain(s string) string { return tags.ReplaceAllString(s, "") }
func Scan(d model.Document, maxLine int, cps float64) []model.Issue {
	out := append([]model.Issue{}, d.ParseIssues...)
	add := func(code string, i int, msg string) {
		out = append(out, model.Issue{Code: code, Entry: i + 1, Message: msg, Severity: "warning"})
	}
	exact := map[string]bool{}
	times := map[string]bool{}
	type interval struct {
		s, e int64
		i    int
	}
	intervals := []interval{}
	for i, e := range d.Entries {
		dur := e.End - e.Start
		if e.Start < 0 || e.End < 0 {
			add("negative_time", i, "Negative timestamp")
		}
		if dur < 0 {
			add("negative_duration", i, "End precedes start")
		} else if dur == 0 {
			add("zero_duration", i, "Zero duration")
		} else if dur < 500 {
			add("short_duration", i, "Duration under 500 ms")
		} else if dur > 10000 {
			add("long_duration", i, "Duration exceeds 10 seconds")
		}
		if strings.TrimSpace(Plain(e.Text)) == "" {
			add("empty", i, "Empty subtitle")
		}
		key := fmt.Sprintf("%d/%d", e.Start, e.End)
		if times[key] {
			add("duplicate_timestamp", i, "Duplicated timestamps")
		}
		times[key] = true
		key = model.DuplicateKey(e)
		if !e.Invalid && exact[key] {
			add("duplicate", i, "Exact duplicate")
		}
		exact[key] = true
		if d.Format == "srt" && e.Index != i+1 {
			add("numbering", i, "SRT numbering is not sequential")
		}
		if i > 0 && e.Start < d.Entries[i-1].Start {
			add("ordering", i, "Entry is out of chronological order")
		}
		for _, line := range strings.Split(Plain(e.Text), "\n") {
			if utf8.RuneCountInString(line) > maxLine {
				add("long_line", i, "Line exceeds configured character limit")
				break
			}
		}
		if dur > 0 && float64(utf8.RuneCountInString(strings.ReplaceAll(Plain(e.Text), "\n", "")))*1000/float64(dur) > cps {
			add("cps", i, "Reading speed exceeds CPS threshold")
		}
		if Suspicious(e.Text) {
			add("tags", i, "Unbalanced or unsupported formatting tags; review manually")
		}
		if !e.Invalid && dur > 0 {
			intervals = append(intervals, interval{e.Start, e.End, i})
		}
	}
	sort.SliceStable(intervals, func(i, j int) bool { return intervals[i].s < intervals[j].s })
	var end int64 = -1
	for _, v := range intervals {
		if v.s < end {
			add("overlap", v.i, "Overlaps an earlier subtitle")
		}
		if v.e > end {
			end = v.e
		}
	}
	return out
}
func Suspicious(s string) bool {
	if strings.Count(s, "{") != strings.Count(s, "}") || strings.Count(s, "<") != strings.Count(s, ">") {
		return true
	}
	stack := []string{}
	for _, tag := range htmlTags.FindAllString(s, -1) {
		x := strings.ToLower(strings.Trim(tag, "<>"))
		if strings.HasPrefix(x, "/") {
			x = strings.TrimPrefix(x, "/")
			if len(stack) == 0 || stack[len(stack)-1] != x {
				return true
			}
			stack = stack[:len(stack)-1]
		} else {
			parts := strings.Fields(x)
			if len(parts) == 0 {
				return true
			}
			x = parts[0]
			if x != "b" && x != "i" && x != "u" && x != "font" && x != "c" && x != "v" && x != "ruby" && x != "rt" {
				return true
			}
			stack = append(stack, x)
		}
	}
	return len(stack) != 0
}
