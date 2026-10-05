package repair

import (
	"regexp"
	"strings"
)

var htmlDelimiter = regexp.MustCompile(`<[^>]*>`)

// CleanHTML keeps balanced known formatting, removes unmatched/unsupported
// delimiters, and never edits the dialogue between them.
func CleanHTML(s string) string {
	type opening struct {
		name  string
		token int
	}
	indices := htmlDelimiter.FindAllStringIndex(s, -1)
	keep := make([]bool, len(indices))
	stack := []opening{}
	for i, pos := range indices {
		tag := strings.TrimSpace(strings.ToLower(s[pos[0]+1 : pos[1]-1]))
		closing := strings.HasPrefix(tag, "/")
		tag = strings.TrimPrefix(tag, "/")
		parts := strings.Fields(tag)
		if len(parts) == 0 {
			continue
		}
		name := parts[0]
		switch name {
		case "b", "i", "u", "font", "c", "v", "ruby", "rt":
		default:
			continue
		}
		if closing {
			if len(stack) > 0 && stack[len(stack)-1].name == name {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				keep[top.token] = true
				keep[i] = true
			}
		} else {
			stack = append(stack, opening{name, i})
		}
	}
	var b strings.Builder
	at := 0
	for i, pos := range indices {
		b.WriteString(s[at:pos[0]])
		if keep[i] {
			b.WriteString(s[pos[0]:pos[1]])
		}
		at = pos[1]
	}
	b.WriteString(s[at:])
	return b.String()
}
