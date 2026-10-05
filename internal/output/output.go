package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func safeSuffix(s string) bool {
	return s != "" && !strings.ContainsAny(s, `<>:"/\|?*`) && !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, " ")
}
func Candidate(source, folder, suffix, format string, n int) (string, error) {
	if !safeSuffix(suffix) {
		return "", fmt.Errorf("suffix must be nonempty and contain no Windows filename special characters")
	}
	if format != "srt" && format != "vtt" && format != "ass" && format != "ssa" {
		return "", fmt.Errorf("unsupported output format")
	}
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source)) + suffix
	if n > 1 {
		base += fmt.Sprintf(" (%d)", n)
	}
	return filepath.Join(folder, base+"."+format), nil
}
func Save(source, folder, suffix, format string, data []byte, replace, confirmed bool) (string, error) {
	if replace {
		if !confirmed {
			return "", fmt.Errorf("replacing the original requires explicit confirmation")
		}
		if !strings.EqualFold(filepath.Ext(source), "."+format) {
			return "", fmt.Errorf("replace original requires the same file format")
		}
		original, err := os.ReadFile(source)
		if err != nil {
			return "", err
		}
		backup := ""
		created := false
		for n := 1; n < 10000; n++ {
			backup = source + ".bak"
			if n > 1 {
				backup += fmt.Sprintf(".%d", n)
			}
			f, e := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if os.IsExist(e) {
				continue
			}
			if e != nil {
				return "", e
			}
			_, e = f.Write(original)
			if e == nil {
				e = f.Sync()
			}
			ce := f.Close()
			if e == nil {
				e = ce
			}
			if e != nil {
				return "", fmt.Errorf("backup failed: %w", e)
			}
			created = true
			break
		}
		if !created {
			return "", fmt.Errorf("unable to create backup")
		}
		f, e := os.CreateTemp(filepath.Dir(source), ".subtitle-doctor-*")
		if e != nil {
			return "", e
		}
		tmp := f.Name()
		defer os.Remove(tmp)
		_, e = f.Write(data)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
		current, e := os.ReadFile(source)
		if e != nil {
			return "", e
		}
		if !bytes.Equal(current, original) {
			return "", fmt.Errorf("source changed while saving; backup retained at %s", backup)
		}
		if e = os.Rename(tmp, source); e != nil {
			return "", fmt.Errorf("replace failed; backup retained at %s: %w", backup, e)
		}
		return source, nil
	}
	if folder == "" {
		return "", fmt.Errorf("select an output folder first")
	}
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}
	for n := 1; n < 10000; n++ {
		path, e := Candidate(source, folder, suffix, format, n)
		if e != nil {
			return "", e
		}
		a, _ := filepath.Abs(source)
		b, _ := filepath.Abs(path)
		if strings.EqualFold(a, b) {
			return "", fmt.Errorf("output cannot equal source")
		}
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(e) {
			continue
		}
		if e != nil {
			return "", e
		}
		_, e = io.Copy(f, bytes.NewReader(data))
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			_ = os.Remove(path)
			return "", e
		}
		return path, nil
	}
	return "", fmt.Errorf("too many filename collisions")
}
