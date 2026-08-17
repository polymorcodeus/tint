package tint

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadThemeConfigEmptyPath(t *testing.T) {
	_, err := LoadThemeConfig("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestLoadThemeConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	_, err := LoadThemeConfig(path)
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestLoadThemeConfigInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadThemeConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadThemeConfigValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.json")
	data := []byte(`{"palette":{"primary":{"color":"#ff0000"}},"styles":{"foo":{"foreground":"primary"}},"huh":{"focused_title":{"foreground":"primary"}}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadThemeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Palette["primary"].Raw != "#ff0000" {
		t.Errorf("primary color = %q, want #ff0000", cfg.Palette["primary"].Raw)
	}
	if cfg.Styles["foo"].Foreground != "primary" {
		t.Errorf("foo foreground = %q, want primary", cfg.Styles["foo"].Foreground)
	}
	if cfg.Huh["focused_title"].Foreground != "primary" {
		t.Errorf("focused_title foreground = %q, want primary", cfg.Huh["focused_title"].Foreground)
	}
}

func TestColorSpecUnmarshalString(t *testing.T) {
	var c ColorSpec
	if err := json.Unmarshal([]byte(`"#ff0000"`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Raw != "#ff0000" {
		t.Errorf("Raw = %q, want #ff0000", c.Raw)
	}
	if c.IsAdaptive() {
		t.Error("string form should not be adaptive")
	}
}

func TestColorSpecUnmarshalObject(t *testing.T) {
	var c ColorSpec
	if err := json.Unmarshal([]byte(`{"light":"#ff0000","dark":"#00ff00"}`), &c); err != nil {
		t.Fatal(err)
	}
	if !c.IsAdaptive() {
		t.Error("object form should be adaptive")
	}
	if c.Light != "#ff0000" || c.Dark != "#00ff00" {
		t.Errorf("Light/Dark = %q/%q, want #ff0000/#00ff00", c.Light, c.Dark)
	}
}

func TestParseLiteralColor(t *testing.T) {
	for _, tc := range []struct {
		input string
		empty bool
	}{
		{"#ff0000", false},
		{"red", false},
		{"212", false},
		{"none", true},
		{"", true},
	} {
		c := parseLiteralColor(tc.input)
		if tc.empty && c != nil {
			t.Errorf("parseLiteralColor(%q) = %v, want nil", tc.input, c)
		}
		if !tc.empty && c == nil {
			t.Errorf("parseLiteralColor(%q) = nil, want non-nil", tc.input)
		}
	}
}

func TestNewThemeDefaults(t *testing.T) {
	th := NewTheme(nil)
	if th == nil {
		t.Fatal("NewTheme(nil) returned nil")
	}
	if th.Style("highlight").GetForeground() == nil {
		t.Error("default highlight style has no foreground")
	}
}

func TestStyleUnknown(t *testing.T) {
	th := NewTheme(nil)
	s := th.Style("does-not-exist")
	if s.Render("plain") != "plain" {
		t.Error("unknown style should render plain text")
	}
}

func TestColorKnownAndLiteral(t *testing.T) {
	th := NewTheme(nil)
	if th.Color("primary") == nil {
		t.Error("primary color is nil")
	}
	if th.Color("#00ff00") == nil {
		t.Error("literal hex color is nil")
	}
}

func TestLipglossList(t *testing.T) {
	th := NewTheme(nil)
	out := LipglossList(th.Style("text"), []string{"one", "two"})
	if !strings.Contains(out, "one") {
		t.Error("list output missing 'one'")
	}
	if !strings.Contains(out, "two") {
		t.Error("list output missing 'two'")
	}
	if !strings.Contains(out, DefaultListBullet) {
		t.Error("list output missing bullet")
	}
}
