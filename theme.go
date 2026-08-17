// Package tint provides an opinionated theme loader and compiler for Go
// CLI/TUI projects built on charmbracelet's lipgloss and huh.
package tint

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// -----------------------------------------------------------------------------
// Configuration Types
// -----------------------------------------------------------------------------

// ColorSpec represents a colour value. In JSON it can be:
//   - A plain string: hex ("#FF5FAF"), ANSI 256 index ("212"), or ANSI name.
//   - An object with "light" and "dark" keys for adaptive colours.
type ColorSpec struct {
	Raw   string `json:"color,omitempty"`
	Light string `json:"light,omitempty"`
	Dark  string `json:"dark,omitempty"`
}

// UnmarshalJSON handles both string and object forms for a ColorSpec.
func (c *ColorSpec) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.Raw = s
		return nil
	}
	type alt ColorSpec
	return json.Unmarshal(data, (*alt)(c))
}

// IsAdaptive reports whether the ColorSpec has light/dark variants.
func (c ColorSpec) IsAdaptive() bool {
	return c.Light != "" || c.Dark != ""
}

// StyleDef defines a reusable lipgloss style. Only non-zero fields are emitted.
type StyleDef struct {
	Foreground       string `json:"foreground,omitempty"`
	Background       string `json:"background,omitempty"`
	BorderForeground string `json:"border_foreground,omitempty"`
	Bold             bool   `json:"bold,omitempty"`
	Italic           bool   `json:"italic,omitempty"`
	Underline        bool   `json:"underline,omitempty"`
	Strikethrough    bool   `json:"strikethrough,omitempty"`
	Faint            bool   `json:"faint,omitempty"`
	Blink            bool   `json:"blink,omitempty"`
	Reverse          bool   `json:"reverse,omitempty"`
}

// ThemeConfig is the raw JSON representation of a user theme.
type ThemeConfig struct {
	Palette map[string]ColorSpec `json:"palette"`
	Styles  map[string]StyleDef  `json:"styles"`
	Huh     map[string]StyleDef  `json:"huh"`
}

// -----------------------------------------------------------------------------
// Compiled Theme (immutable)
// -----------------------------------------------------------------------------

// Theme is a compiled, ready-to-use theme with resolved colors and styles.
type Theme struct {
	hasDarkBg      bool
	colors         map[string]color.Color
	styles         map[string]lipgloss.Style
	huhDefinitions map[string]StyleDef
}

// NewTheme compiles a ThemeConfig into a Theme with resolved colors and styles.
func NewTheme(cfg *ThemeConfig) *Theme {
	if cfg == nil {
		cfg = DefaultThemeConfig()
	}

	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	t := &Theme{
		hasDarkBg:      hasDark,
		colors:         make(map[string]color.Color),
		styles:         make(map[string]lipgloss.Style),
		huhDefinitions: cfg.Huh,
	}

	for name, spec := range cfg.Palette {
		t.colors[name] = t.resolveColorSpec(spec)
	}
	for name, def := range cfg.Styles {
		t.styles[name] = t.buildStyle(def)
	}

	return t
}

// Color looks up a named color from the palette, falling back to parsing the name as a literal color.
func (t *Theme) Color(name string) color.Color {
	if c, ok := t.colors[name]; ok {
		return c
	}
	return parseLiteralColor(name)
}

// Style looks up a named style, returning an empty style if the name is not found.
func (t *Theme) Style(name string) lipgloss.Style {
	if s, ok := t.styles[name]; ok {
		return s
	}
	return lipgloss.NewStyle()
}

// HuhTheme returns a huh.ThemeFunc that overlays custom styles onto a base theme.
func (t *Theme) HuhTheme(interactive bool) huh.ThemeFunc {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		base := huh.ThemeBase(isDark)
		if interactive {
			base = huh.ThemeCharm(isDark)
		}

		apply := func(target *lipgloss.Style, key string) {
			if def, ok := t.huhDefinitions[key]; ok {
				*target = t.overlayStyleDef(*target, def)
			}
		}

		type item struct {
			target *lipgloss.Style
			key    string
		}

		applyAll := func(items []item) {
			for _, it := range items {
				apply(it.target, it.key)
			}
		}

		applyAll([]item{
			{&base.Focused.Title, "focused_title"},
			{&base.Focused.Description, "focused_description"},
			{&base.Focused.SelectedOption, "focused_selected_option"},
			{&base.Focused.UnselectedOption, "focused_unselected_option"},
			{&base.Focused.ErrorIndicator, "focused_error_indicator"},
			{&base.Focused.ErrorMessage, "focused_error_message"},
			{&base.Focused.SelectSelector, "focused_select_selector"},
			{&base.Focused.NextIndicator, "focused_next_indicator"},
			{&base.Focused.PrevIndicator, "focused_prev_indicator"},
			{&base.Focused.FocusedButton, "focused_focused_button"},
			{&base.Focused.BlurredButton, "focused_blurred_button"},
			{&base.Focused.Directory, "focused_directory"},
			{&base.Focused.File, "focused_file"},
			{&base.Focused.Option, "focused_option"},
			{&base.Focused.MultiSelectSelector, "focused_multi_select_selector"},
			{&base.Focused.SelectedPrefix, "focused_selected_prefix"},
			{&base.Focused.UnselectedPrefix, "focused_unselected_prefix"},
			{&base.Focused.Card, "focused_card"},
			{&base.Focused.NoteTitle, "focused_note_title"},
			{&base.Focused.Next, "focused_next"},
		})

		applyAll([]item{
			{&base.Blurred.Title, "blurred_title"},
			{&base.Blurred.Description, "blurred_description"},
			{&base.Blurred.SelectedOption, "blurred_selected_option"},
			{&base.Blurred.UnselectedOption, "blurred_unselected_option"},
			{&base.Blurred.ErrorIndicator, "blurred_error_indicator"},
			{&base.Blurred.ErrorMessage, "blurred_error_message"},
			{&base.Blurred.SelectSelector, "blurred_select_selector"},
			{&base.Blurred.NextIndicator, "blurred_next_indicator"},
			{&base.Blurred.PrevIndicator, "blurred_prev_indicator"},
			{&base.Blurred.FocusedButton, "blurred_focused_button"},
			{&base.Blurred.BlurredButton, "blurred_blurred_button"},
			{&base.Blurred.Directory, "blurred_directory"},
			{&base.Blurred.File, "blurred_file"},
			{&base.Blurred.Option, "blurred_option"},
			{&base.Blurred.MultiSelectSelector, "blurred_multi_select_selector"},
			{&base.Blurred.SelectedPrefix, "blurred_selected_prefix"},
			{&base.Blurred.UnselectedPrefix, "blurred_unselected_prefix"},
			{&base.Blurred.Card, "blurred_card"},
			{&base.Blurred.NoteTitle, "blurred_note_title"},
			{&base.Blurred.Next, "blurred_next"},
		})

		applyAll([]item{
			{&base.Focused.TextInput.Cursor, "focused_textinput_cursor"},
			{&base.Focused.TextInput.CursorText, "focused_textinput_cursor_text"},
			{&base.Focused.TextInput.Placeholder, "focused_textinput_placeholder"},
			{&base.Focused.TextInput.Prompt, "focused_textinput_prompt"},
			{&base.Focused.TextInput.Text, "focused_textinput_text"},
		})

		applyAll([]item{
			{&base.Blurred.TextInput.Cursor, "blurred_textinput_cursor"},
			{&base.Blurred.TextInput.CursorText, "blurred_textinput_cursor_text"},
			{&base.Blurred.TextInput.Placeholder, "blurred_textinput_placeholder"},
			{&base.Blurred.TextInput.Prompt, "blurred_textinput_prompt"},
			{&base.Blurred.TextInput.Text, "blurred_textinput_text"},
		})

		applyAll([]item{
			{&base.Group.Title, "group_title"},
			{&base.Group.Description, "group_description"},
		})

		return base
	})
}

// -----------------------------------------------------------------------------
// Interactive Helpers
// -----------------------------------------------------------------------------

// ConfirmInput presents a huh confirm prompt using the supplied theme.
func ConfirmInput(title, description string, theme huh.ThemeFunc) (bool, error) {
	var confirm bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yep!").
				Negative("Wait, no").
				Value(&confirm).
				WithTheme(theme),
		),
	)

	if err := form.Run(); err != nil {
		return false, err
	}
	return confirm, nil
}

// DefaultListBullet is the glyph used by LipglossList.
const DefaultListBullet = "\U000f1978" // "nf-md-dots_circle"

// LipglossList renders a bulleted list using the supplied lipgloss style.
func LipglossList(gloss lipgloss.Style, items []string) string {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = gloss.Render(DefaultListBullet, s)
	}
	return strings.Join(parts, "\n")
}

// -----------------------------------------------------------------------------
// File Loading
// -----------------------------------------------------------------------------

// LoadThemeConfig reads and parses a theme.json file into a ThemeConfig.
// A missing file is returned as an error so callers can decide whether to
// fall back to defaults.
func LoadThemeConfig(path string) (*ThemeConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("theme file path is empty")
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("load theme %q: %w", path, err)
	}

	var cfg ThemeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse theme %q: %w", path, err)
	}
	return &cfg, nil
}

// -----------------------------------------------------------------------------
// Internal
// -----------------------------------------------------------------------------

// DefaultThemeConfig returns the built-in default theme configuration.
func DefaultThemeConfig() *ThemeConfig {
	return &ThemeConfig{
		Palette: map[string]ColorSpec{
			"primary":          {Light: "#FF80AB", Dark: "#FF4081"},
			"primary_accent":   {Light: "#FF5FAF", Dark: "#FE5F86"},
			"secondary":        {Light: "#c5adf9", Dark: "#7d56f4"},
			"secondary_accent": {Light: "#64FCDA", Dark: "#04b575"},
			"highlight":        {Light: "#f5d76e", Dark: "#ffd640"},
			"error":            {Raw: "#ff5c57"},
			"text":             {Light: "#14121a", Dark: "#f5f1fa"},
		},
		Styles: map[string]StyleDef{
			"primary":          {Foreground: "primary"},
			"primary_accent":   {Foreground: "primary_accent"},
			"secondary":        {Foreground: "secondary"},
			"secondary_accent": {Foreground: "secondary_accent"},
			"highlight":        {Foreground: "highlight", Bold: true},
			"error":            {Foreground: "error", Bold: true},
			"text":             {Foreground: "text"},
			"dimmed":           {Foreground: "243"},
			"help":             {Foreground: "240"},
		},
		Huh: map[string]StyleDef{
			"focused_title":           {Foreground: "primary", Bold: true},
			"focused_selected_option": {Foreground: "secondary_accent"},
			"focused_description":     {Foreground: "243", Italic: true},
			"blurred_description":     {Foreground: "243", Italic: true},
		},
	}
}

func (t *Theme) resolveColorSpec(spec ColorSpec) color.Color {
	if spec.IsAdaptive() {
		light := parseLiteralColor(spec.Light)
		dark := parseLiteralColor(spec.Dark)
		return lipgloss.LightDark(t.hasDarkBg)(light, dark)
	}
	return parseLiteralColor(spec.Raw)
}

func parseLiteralColor(v string) color.Color {
	v = strings.ToLower(strings.TrimSpace(v))

	switch v {
	case "black":
		return lipgloss.Black
	case "red":
		return lipgloss.Red
	case "green":
		return lipgloss.Green
	case "yellow":
		return lipgloss.Yellow
	case "blue":
		return lipgloss.Blue
	case "magenta":
		return lipgloss.Magenta
	case "cyan":
		return lipgloss.Cyan
	case "white":
		return lipgloss.White
	case "brightblack", "bright_black", "gray", "grey":
		return lipgloss.BrightBlack
	case "brightred", "bright_red":
		return lipgloss.BrightRed
	case "brightgreen", "bright_green":
		return lipgloss.BrightGreen
	case "brightyellow", "bright_yellow":
		return lipgloss.BrightYellow
	case "brightblue", "bright_blue":
		return lipgloss.BrightBlue
	case "brightmagenta", "bright_magenta":
		return lipgloss.BrightMagenta
	case "brightcyan", "bright_cyan":
		return lipgloss.BrightCyan
	case "brightwhite", "bright_white":
		return lipgloss.BrightWhite
	case "none", "":
		return nil
	}

	return lipgloss.Color(v)
}

func (t *Theme) parseColorValue(v string) color.Color {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "none") {
		return nil
	}
	if c, ok := t.colors[v]; ok {
		return c
	}
	return parseLiteralColor(v)
}

func (t *Theme) buildStyle(def StyleDef) lipgloss.Style {
	return t.overlayStyleDef(lipgloss.NewStyle(), def)
}

func (t *Theme) overlayStyleDef(base lipgloss.Style, def StyleDef) lipgloss.Style {
	s := base
	if def.Foreground != "" {
		s = s.Foreground(t.parseColorValue(def.Foreground))
	}
	if def.Background != "" {
		s = s.Background(t.parseColorValue(def.Background))
	}
	if def.BorderForeground != "" {
		s = s.BorderForeground(t.parseColorValue(def.BorderForeground))
	}
	if def.Bold {
		s = s.Bold(true)
	}
	if def.Italic {
		s = s.Italic(true)
	}
	if def.Underline {
		s = s.Underline(true)
	}
	if def.Strikethrough {
		s = s.Strikethrough(true)
	}
	if def.Faint {
		s = s.Faint(true)
	}
	if def.Blink {
		s = s.Blink(true)
	}
	if def.Reverse {
		s = s.Reverse(true)
	}
	return s
}
