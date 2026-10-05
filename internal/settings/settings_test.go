package settings

import (
	"testing"
)

func TestPersistence(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	s := Default()
	s.OutputFolder = `C:\Subtitles Ștefan\Output`
	s.Suffix = "_repaired"
	s.Format = "vtt"
	s.MaxChars = 50
	if e := Save(s); e != nil {
		t.Fatal(e)
	}
	got, e := Load()
	if e != nil || got != s {
		t.Fatal(got, e)
	}
	s.Overwrite = true
	s.Backup = false
	if e := Save(s); e == nil {
		t.Fatal("allowed overwrite without backup")
	}
}
