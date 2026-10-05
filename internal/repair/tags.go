package repair

import "github.com/AlinTibi/SubtitleDoctor/internal/markup"

func CleanHTML(s string) string { return markup.Clean(s) }
