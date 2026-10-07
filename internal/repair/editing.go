package repair

import (
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/markup"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"strings"
)

func editable(d model.Document, e model.Entry) error {
	if e.Invalid || e.Start < 0 || e.End <= e.Start {
		return fmt.Errorf("invalid cue or timing; correct it before editing")
	}
	if strings.TrimSpace(markup.StripHTML(e.Text)) == "" {
		return fmt.Errorf("empty cue cannot be split or merged")
	}
	if markup.Suspicious(e.Text) {
		return fmt.Errorf("malformed or unsupported formatting; review before editing")
	}
	if d.Format == "vtt" && len(e.Fields) > 0 && e.Fields[0] != "" {
		return fmt.Errorf("cue identifiers cannot be preserved by this operation; edit these cues manually")
	}
	if (d.Format == "ass" || d.Format == "ssa") && len(e.Fields) != len(d.EventFormat) {
		return fmt.Errorf("incomplete ASS/SSA metadata; edit manually")
	}
	return nil
}

// EditCues is transactional: any incompatible cue leaves the draft unchanged.
func EditCues(d *model.Document, ids []int, action string) error {
	if len(ids) == 0 {
		return fmt.Errorf("select entries")
	}
	for i, id := range ids {
		if id < 0 || id >= len(d.Entries) || (i > 0 && id != ids[i-1]+1) {
			return fmt.Errorf("select contiguous entries in order")
		}
		if err := editable(*d, d.Entries[id]); err != nil {
			return err
		}
	}
	first := ids[0]
	e := d.Entries[first]
	var replacement []model.Entry
	switch action {
	case "split":
		if len(ids) != 1 {
			return fmt.Errorf("select one entry to split")
		}
		unit := int64(1)
		if d.Format == "ass" || d.Format == "ssa" {
			unit = 10
			if e.Start%unit != 0 || e.End%unit != 0 {
				return fmt.Errorf("ASS/SSA split requires centisecond-aligned timing")
			}
		}
		if e.End-e.Start < 2*unit {
			return fmt.Errorf("duration is too short for two valid cues")
		}
		left, right, err := markup.Split(e.Text)
		if err != nil {
			return err
		}
		mid := e.Start + ((e.End-e.Start)/2/unit)*unit
		a, b := e, e
		a.End = mid
		a.Text = left
		b.Start = mid
		b.Text = right
		replacement = []model.Entry{a, b}
	case "merge":
		if len(ids) < 2 {
			return fmt.Errorf("select at least two entries")
		}
		for _, id := range ids {
			if !markup.MergeSafe(d.Entries[id].Text) {
				return fmt.Errorf("stateful ASS overrides, implicit voice spans or karaoke timestamps cannot be safely merged; edit manually")
			}
		}
		metadata := func(v model.Entry) string { v.Start = 0; v.End = 0; v.Text = ""; return model.DuplicateKey(*d, v) }
		text := []string{e.Text}
		for i := 1; i < len(ids); i++ {
			prev, current := d.Entries[ids[i-1]], d.Entries[ids[i]]
			if prev.End != current.Start {
				return fmt.Errorf("merge requires consecutive timing without gaps or overlaps; edit timing explicitly first")
			}
			if metadata(e) != metadata(current) {
				return fmt.Errorf("incompatible cue positioning/style/settings metadata; merge refused")
			}
			text = append(text, current.Text)
		}
		e.End = d.Entries[ids[len(ids)-1]].End
		e.Text = strings.Join(text, "\n")
		if markup.Suspicious(e.Text) {
			return fmt.Errorf("formatting cannot be safely merged")
		}
		replacement = []model.Entry{e}
	default:
		return fmt.Errorf("unknown cue edit")
	}
	out := append([]model.Entry(nil), d.Entries[:first]...)
	out = append(out, replacement...)
	out = append(out, d.Entries[ids[len(ids)-1]+1:]...)
	d.Entries = out
	return nil
}
