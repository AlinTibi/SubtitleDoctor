package markup

import (
	"fmt"
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

type boundary struct{ start, end int }

// Boundaries are whitespace in dialogue only, never in attributes, commands or entities.
func boundaries(s string) []boundary {
	protectedSpans := protected.FindAllStringIndex(s, -1)
	var out []boundary
	at := 0
	scan := func(end int) {
		for at < end {
			r, size := utf8.DecodeRuneInString(s[at:end])
			if !unicode.IsSpace(r) {
				at += size
				continue
			}
			start := at
			for at < end {
				r, size = utf8.DecodeRuneInString(s[at:end])
				if !unicode.IsSpace(r) {
					break
				}
				at += size
			}
			out = append(out, boundary{start, at})
		}
	}
	for _, p := range protectedSpans {
		scan(p[0])
		at = p[1]
	}
	scan(len(s))
	return out
}

func visible(s string) int {
	s = protected.ReplaceAllStringFunc(s, func(token string) string {
		if strings.HasPrefix(token, "{") {
			return ""
		}
		return token
	})
	return utf8.RuneCountInString(html.UnescapeString(StripHTML(s)))
}

// Wrap keeps existing line breaks and every delimiter verbatim. Inserted breaks
// are dialogue whitespace; spanning formatting remains balanced within the cue.
func Wrap(s string, width, maxLines int) (string, error) {
	if Suspicious(s) {
		return "", fmt.Errorf("malformed or unsupported markup; review it before wrapping")
	}
	eol := "\n"
	if strings.Contains(s, "\r\n") {
		eol = "\r\n"
	}
	var out strings.Builder
	start, column, lines := 0, 0, 1
	for _, p := range append(boundaries(s), boundary{len(s), len(s)}) {
		word := s[start:p.start]
		n := visible(word)
		if n > width {
			return "", fmt.Errorf("a dialogue word exceeds the line limit")
		}
		if column > 0 && n > 0 && column+1+n > width {
			out.WriteString(eol)
			column = 0
			lines++
		} else if column > 0 && n > 0 {
			out.WriteByte(' ')
			column++
		}
		out.WriteString(word)
		column += n
		separator := s[p.start:p.end]
		if strings.Contains(separator, "\n") || strings.Contains(separator, "\r") {
			breaks := strings.Count(strings.ReplaceAll(separator, "\r\n", "\n"), "\n")
			if breaks == 0 {
				breaks = 1
			}
			out.WriteString(strings.Repeat(eol, breaks))
			column = 0
			lines += breaks
		}
		start = p.end
	}
	if lines > maxLines {
		return "", fmt.Errorf("needs %d lines; raise the max lines limit", lines)
	}
	return out.String(), nil
}

// Split closes and reopens simple formatting at a dialogue word boundary.
// Stateful ASS overrides, ruby and karaoke timestamps require manual editing.
func Split(s string) (string, string, error) {
	if Suspicious(s) || strings.ContainsAny(s, "{}") {
		return "", "", fmt.Errorf("split requires balanced HTML/WebVTT formatting without ASS overrides")
	}
	for _, p := range htmlTokens.FindAllStringIndex(s, -1) {
		name, _, _, _ := tag(s[p[0]:p[1]])
		if name == "ruby" || name == "rt" || name == "timestamp" || name == "br" {
			return "", "", fmt.Errorf("split of ruby, timestamp or line-break tags requires manual editing")
		}
	}
	var candidates []boundary
	for _, p := range boundaries(s) {
		if visible(strings.TrimSpace(s[:p.start])) > 0 && visible(strings.TrimSpace(s[p.end:])) > 0 {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("entry needs at least two dialogue words")
	}
	p := candidates[len(candidates)/2]
	left, right := strings.TrimSpace(s[:p.start]), strings.TrimSpace(s[p.end:])
	var stack []string
	for _, pos := range htmlTokens.FindAllStringIndex(left, -1) {
		token := left[pos[0]:pos[1]]
		_, closing, _, void := tag(token)
		if void {
			continue
		}
		if closing {
			if len(stack) == 0 {
				return "", "", fmt.Errorf("unbalanced formatting")
			}
			stack = stack[:len(stack)-1]
		} else {
			stack = append(stack, token)
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		name, _, _, _ := tag(stack[i])
		left += "</" + name + ">"
	}
	right = strings.Join(stack, "") + right
	if Suspicious(left) || Suspicious(right) {
		return "", "", fmt.Errorf("formatting cannot be safely split")
	}
	return left, right, nil
}

// Implicit voice closure is valid in an individual VTT cue, but concatenating
// another cue would extend that voice span over unrelated dialogue.
func MergeSafe(s string) bool {
	voices := 0
	for _, p := range htmlTokens.FindAllStringIndex(s, -1) {
		name, closing, _, _ := tag(s[p[0]:p[1]])
		if name == "timestamp" {
			return false
		}
		if name == "v" {
			if closing {
				voices--
			} else {
				voices++
			}
		}
	}
	return voices == 0 && !Suspicious(s) && !strings.ContainsAny(s, "{}")
}
