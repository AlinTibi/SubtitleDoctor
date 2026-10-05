package settings

import (
	"encoding/json"
	"fmt"
	"github.com/AlinTibi/SubtitleDoctor/internal/model"
	"os"
	"path/filepath"
)

type Settings struct {
	OutputFolder string  `json:"outputFolder"`
	Suffix       string  `json:"suffix"`
	Encoding     string  `json:"encoding"`
	Format       string  `json:"format"`
	MaxChars     int     `json:"maxChars"`
	MaxLines     int     `json:"maxLines"`
	CPS          float64 `json:"cps"`
	Backup       bool    `json:"backup"`
	Overwrite    bool    `json:"overwrite"`
}

func Default() Settings {
	return Settings{Suffix: "_fixed", Encoding: "UTF-8", Format: "source", MaxChars: 42, MaxLines: 2, CPS: 20, Backup: true}
}
func Path() (string, error) {
	p, e := os.UserConfigDir()
	return filepath.Join(p, "SubtitleDoctor", "settings.json"), e
}
func Validate(s Settings) error {
	if s.MaxChars < 10 || s.MaxChars > 500 || s.MaxLines < 1 || s.MaxLines > 20 || s.CPS < 1 || s.CPS > 1000 {
		return fmt.Errorf("invalid line or reading-speed limits")
	}
	if !model.ValidSuffix(s.Suffix) {
		return fmt.Errorf("invalid output suffix")
	}
	if s.Encoding != "UTF-8" && s.Encoding != "UTF-16LE" && s.Encoding != "Windows-1252" {
		return fmt.Errorf("invalid encoding")
	}
	if s.Format != "source" && s.Format != "srt" && s.Format != "vtt" && s.Format != "ass" && s.Format != "ssa" {
		return fmt.Errorf("invalid default format")
	}
	if !s.Backup {
		return fmt.Errorf("backups are mandatory when replacing originals")
	}
	return nil
}
func Load() (Settings, error) {
	s := Default()
	p, e := Path()
	if e != nil {
		return s, e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(b, &s); e != nil {
		return Default(), e
	}
	if e = Validate(s); e != nil {
		return Default(), e
	}
	return s, nil
}
func Save(s Settings) error {
	if e := Validate(s); e != nil {
		return e
	}
	p, e := Path()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), "settings-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(tmp, p)
}
