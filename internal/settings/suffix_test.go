package settings

import (
	"github.com/AlinTibi/SubtitleDoctor/internal/output"
	"testing"
)

func TestSuffixPersistenceMatchesOutput(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	good := Default()
	if err := Save(good); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "_bad.", "_bad ", "_bad/", "_bad\\", "_bad:", "_bad*", "_bad?", "_bad\"", "_bad<", "_bad>", "_bad|", "_bad\x00", "_bad\n"} {
		s := good
		s.Suffix = suffix
		if err := Validate(s); err == nil {
			t.Errorf("validated %q", suffix)
		}
		if err := Save(s); err == nil {
			t.Errorf("persisted %q", suffix)
		}
		if _, err := output.Candidate("movie.srt", t.TempDir(), suffix, "srt", 1); err == nil {
			t.Errorf("output accepted %q", suffix)
		}
		loaded, err := Load()
		if err != nil || loaded != good {
			t.Fatal("invalid save changed settings", loaded, err)
		}
	}
	for _, suffix := range []string{"_fixed", "_Ștefan", "_with space", "_v1.2"} {
		s := good
		s.Suffix = suffix
		if err := Save(s); err != nil {
			t.Fatal(err)
		}
		if _, err := output.Candidate("movie.srt", t.TempDir(), suffix, "srt", 1); err != nil {
			t.Fatal(err)
		}
	}
}
