// Package markup handles subtitle delimiters without rewriting their contents.
package markup

import (
	"regexp"
	"strings"
)

var htmlTokens = regexp.MustCompile(`<(?:[^>"']|"[^"]*"|'[^']*')*>`)
var protected = regexp.MustCompile(`<(?:[^>"']|"[^"]*"|'[^']*')*>|\{[^}]*\}|&(?:#[0-9]+|#x[0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);|\\[Nnh]`)
var timestamp = regexp.MustCompile(`^(?:[0-9]+:)?[0-5][0-9]:[0-5][0-9]\.[0-9]{3}$`)

func tag(token string) (name string, closing, known, void bool) {
	s := strings.TrimSpace(token[1 : len(token)-1])
	closing = strings.HasPrefix(s, "/")
	s = strings.TrimSpace(strings.TrimPrefix(s, "/"))
	if timestamp.MatchString(s) && !closing {
		return "timestamp", false, true, true
	}
	parts := strings.Fields(strings.TrimSuffix(s, "/"))
	if len(parts) == 0 {
		return "", closing, false, false
	}
	name = strings.ToLower(strings.SplitN(parts[0], ".", 2)[0])
	switch name {
	case "br":
		return name, closing, !closing, true
	case "b", "i", "u", "font", "c", "v", "ruby", "rt", "lang":
		return name, closing, true, false
	}
	return name, closing, false, false
}

// WebVTT allows the final rt end tag and a sole voice span end tag to be omitted.
// See https://www.w3.org/TR/webvtt1/#webvtt-cue-text.
func inspect(s string) (indices [][]int, keep []bool, suspicious bool) {
	indices = htmlTokens.FindAllStringIndex(s, -1)
	keep = make([]bool, len(indices))
	type opening struct {
		name  string
		token int
	}
	stack := []opening{}
	at := 0
	for i, pos := range indices {
		if strings.ContainsAny(s[at:pos[0]], "<>") {
			suspicious = true
		}
		at = pos[1]
		name, closing, known, void := tag(s[pos[0]:pos[1]])
		if !known {
			keep[i] = true
			suspicious = true
			continue
		}
		if void {
			keep[i] = true
			continue
		}
		if closing {
			if name == "ruby" && len(stack) > 0 && stack[len(stack)-1].name == "rt" {
				keep[stack[len(stack)-1].token] = true
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 && stack[len(stack)-1].name == name {
				keep[stack[len(stack)-1].token], keep[i] = true, true
				stack = stack[:len(stack)-1]
			} else {
				suspicious = true
			}
		} else {
			if name == "rt" && (len(stack) == 0 || stack[len(stack)-1].name != "ruby") {
				suspicious = true
			}
			stack = append(stack, opening{name, i})
		}
	}
	if strings.ContainsAny(s[at:], "<>") {
		suspicious = true
	}
	if len(stack) == 1 && stack[0].name == "v" && strings.TrimSpace(s[:indices[stack[0].token][0]]) == "" {
		keep[stack[0].token] = true
		stack = nil
	}
	if len(stack) > 0 {
		suspicious = true
	}
	return
}

func Suspicious(s string) bool {
	_, _, bad := inspect(s)
	return bad || strings.Count(s, "{") != strings.Count(s, "}")
}

// Clean removes only unmatched recognized delimiters. Unknown syntax is retained
// for manual review; line-break tokens and all dialogue stay intact.
func Clean(s string) string {
	indices, keep, _ := inspect(s)
	var b strings.Builder
	at := 0
	for i, p := range indices {
		b.WriteString(s[at:p[0]])
		if keep[i] {
			b.WriteString(s[p[0]:p[1]])
		}
		at = p[1]
	}
	b.WriteString(s[at:])
	return b.String()
}

func StripHTML(s string) string {
	return htmlTokens.ReplaceAllStringFunc(s, func(token string) string {
		name, _, _, _ := tag(token)
		if name == "br" {
			return "\n"
		}
		return ""
	})
}

// MapDialogue preserves markup, entities and ASS escapes byte-for-byte. Callback
// state crosses spans, so formatting delimiters cannot reset sentence case.
func MapDialogue(s string, transform func(string) string) string {
	var b strings.Builder
	at := 0
	for _, p := range protected.FindAllStringIndex(s, -1) {
		b.WriteString(transform(s[at:p[0]]))
		b.WriteString(s[p[0]:p[1]])
		at = p[1]
	}
	b.WriteString(transform(s[at:]))
	return b.String()
}
