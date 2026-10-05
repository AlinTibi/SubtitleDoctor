package output

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestPreservationCollisionsAndBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "film Ștefan.srt")
	original := []byte("original")
	if e := os.WriteFile(src, original, 0644); e != nil {
		t.Fatal(e)
	}
	for n := 1; n <= 3; n++ {
		p, e := Save(src, dir, "_fixed", "srt", []byte("changed"), false, false)
		expected, _ := Candidate(src, dir, "_fixed", "srt", n)
		if e != nil || p != expected {
			t.Fatal(p, e, expected)
		}
	}
	b, _ := os.ReadFile(src)
	if !bytes.Equal(b, original) {
		t.Fatal("source changed")
	}
	if _, e := Save(src, dir, "_fixed", "srt", []byte("new"), true, false); e == nil {
		t.Fatal("confirmation bypassed")
	}
	if _, e := Save(src, dir, "_fixed", "srt", []byte("new"), true, true); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(src + ".bak")
	if !bytes.Equal(b, original) {
		t.Fatal("invalid backup")
	}
	if _, e := Save(src, dir, "_fixed", "srt", []byte("newer"), true, true); e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(src + ".bak.2")
	if string(b) != "new" {
		t.Fatal("backup overwritten")
	}
}
func TestConcurrentCollisions(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	paths := sync.Map{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := Save(filepath.Join(dir, "movie.srt"), dir, "_fixed", "srt", []byte("test"), false, false)
			if e != nil {
				t.Error(e)
				return
			}
			if _, dup := paths.LoadOrStore(p, true); dup {
				t.Error("collision", p)
			}
		}()
	}
	wg.Wait()
}
func TestUnsafeSuffix(t *testing.T) {
	for _, s := range []string{"", "../x", `\x`, "x:", "x."} {
		if _, e := Candidate("movie.srt", "out", s, "srt", 1); e == nil {
			t.Error("accepted suffix", s)
		}
	}
}
