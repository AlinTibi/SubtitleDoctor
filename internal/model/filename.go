package model

import "strings"

// ValidSuffix is shared by settings persistence and output filename creation.
func ValidSuffix(s string) bool {
	if s == "" || strings.ContainsAny(s, `<>:"/\|?*`) || strings.HasSuffix(s, ".") || strings.HasSuffix(s, " ") {
		return false
	}
	for _, r := range s {
		if r < 32 {
			return false
		}
	}
	return true
}
