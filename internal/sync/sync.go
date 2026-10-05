package sync

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"math"
)

func Linear(d *model.Document, scale, offset float64) error {
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 || math.IsNaN(offset) || math.IsInf(offset, 0) {
		return fmt.Errorf("timing correction must be finite and forward-moving")
	}
	next := model.Clone(*d)
	for i := range next.Entries {
		e := &next.Entries[i]
		if e.Invalid {
			return fmt.Errorf("entry %d has an invalid timestamp", i+1)
		}
		a := float64(e.Start)*scale + offset
		b := float64(e.End)*scale + offset
		if a < 0 || b < 0 || a > 1e12 || b > 1e12 {
			return fmt.Errorf("entry %d would move outside 0–1,000,000,000 seconds; no changes applied", i+1)
		}
		e.Start = int64(math.Round(a))
		e.End = int64(math.Round(b))
	}
	*d = next
	return nil
}
func Offset(d *model.Document, ms float64) error { return Linear(d, 1, ms) }
func TwoPoint(d *model.Document, a, b, c, e float64) error {
	if c <= a || e <= b {
		return fmt.Errorf("second reference times must follow the first references")
	}
	scale := (e - b) / (c - a)
	return Linear(d, scale, b-a*scale)
}
func FPS(d *model.Document, from, to float64) error {
	if from <= 0 || to <= 0 {
		return fmt.Errorf("FPS values must be positive")
	}
	return Linear(d, from/to, 0)
}
