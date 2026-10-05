package sync

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"testing"
)

func doc() model.Document {
	return model.Document{Entries: []model.Entry{{Start: 3000, End: 5000, Text: "hello"}}}
}
func TestOffset(t *testing.T) {
	d := doc()
	if e := Offset(&d, 1500); e != nil || d.Entries[0].Start != 4500 {
		t.Fatal(d, e)
	}
	if e := Offset(&d, -2000); e != nil || d.Entries[0].Start != 2500 {
		t.Fatal(d, e)
	}
	if e := Offset(&d, -4000); e == nil || d.Entries[0].Start != 2500 {
		t.Fatal("negative offset must be transactional")
	}
}
func TestTwoPoint(t *testing.T) {
	d := doc()
	if e := TwoPoint(&d, 1000, 2000, 5000, 10000); e != nil || d.Entries[0].Start != 6000 || d.Entries[0].End != 10000 {
		t.Fatal(d, e)
	}
	if e := TwoPoint(&d, 1, 2, 1, 4); e == nil {
		t.Fatal("accepted degenerate references")
	}
}
func TestFPS(t *testing.T) {
	d := doc()
	if e := FPS(&d, 25, 24); e != nil || d.Entries[0].Start != 3125 {
		t.Fatal(d, e)
	}
	if e := FPS(&d, 0, 25); e == nil {
		t.Fatal("accepted zero FPS")
	}
}
