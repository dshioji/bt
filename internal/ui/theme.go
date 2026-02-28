package ui

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

// ThemeColors holds all configurable color values as hex strings (#RRGGBB).
// Use in ~/.config/bt/mytheme.yaml and set `theme: mytheme` in conf.yaml.
type ThemeColors struct {
	SelectedPath           string `mapstructure:"selected_path"`
	FinfoPermissions       string `mapstructure:"finfo_permissions"`
	FinfoLastUpdated       string `mapstructure:"finfo_last_updated"`
	FinfoSize              string `mapstructure:"finfo_size"`
	FinfoSep               string `mapstructure:"finfo_sep"`
	OperationBar           string `mapstructure:"operation_bar"`
	OperationBarInputBg    string `mapstructure:"operation_bar_input_bg"`
	ErrBar                 string `mapstructure:"err_bar"`
	HelpMsg                string `mapstructure:"help_msg"`
	HelpContent            string `mapstructure:"help_content"`
	TreeRegularFile        string `mapstructure:"tree_regular_file"`
	TreeDirectory          string `mapstructure:"tree_directory"`
	TreeLink               string `mapstructure:"tree_link"`
	TreeMarkedBg           string `mapstructure:"tree_marked_bg"`
	TreeSelectionArrow     string `mapstructure:"tree_selection_arrow"`
	TreeIndent             string `mapstructure:"tree_indent"`
	TreeIndentSelected     string `mapstructure:"tree_indent_selected"`
	PlainTextPreview       string `mapstructure:"plain_text_preview"`
	PlainTextPreviewBorder string `mapstructure:"plain_text_preview_border"`
}

// DarkThemeColors is the built-in dark theme (terminal dark background).
var DarkThemeColors = ThemeColors{
	SelectedPath:           "#74AC6D",
	FinfoPermissions:       "#ACA46D",
	FinfoLastUpdated:       "#E6E6E6",
	FinfoSize:              "#E6E6E6",
	FinfoSep:               "#2b2b2b",
	OperationBar:           "#E6E6E6",
	OperationBarInputBg:    "#3C3C3C",
	ErrBar:                 "#AC6D74",
	HelpMsg:                "#ACA46D",
	HelpContent:            "#8c7ca6",
	TreeRegularFile:        "#E6E6E6",
	TreeDirectory:          "#6D74AC",
	TreeLink:               "#6DACA4",
	TreeMarkedBg:           "#363636",
	TreeSelectionArrow:     "#ACA46D",
	TreeIndent:             "#363636",
	TreeIndentSelected:     "#ACA46D",
	PlainTextPreview:       "#a8a8a8",
	PlainTextPreviewBorder: "#363636",
}

// LightThemeColors is the built-in light theme for light terminal backgrounds.
var LightThemeColors = ThemeColors{
	SelectedPath:           "#2E7D32",
	FinfoPermissions:       "#E65100",
	FinfoLastUpdated:       "#212121",
	FinfoSize:              "#212121",
	FinfoSep:               "#9E9E9E",
	OperationBar:           "#212121",
	OperationBarInputBg:    "#E0E0E0",
	ErrBar:                 "#B71C1C",
	HelpMsg:                "#E65100",
	HelpContent:            "#4527A0",
	TreeRegularFile:        "#212121",
	TreeDirectory:          "#1565C0",
	TreeLink:               "#00695C",
	TreeMarkedBg:           "#F5F5F5",
	TreeSelectionArrow:     "#E65100",
	TreeIndent:             "#BDBDBD",
	TreeIndentSelected:     "#E65100",
	PlainTextPreview:       "#424242",
	PlainTextPreviewBorder: "#BDBDBD",
}

// ToStylesheet converts ThemeColors into a Stylesheet for the renderer.
func (tc ThemeColors) ToStylesheet() Stylesheet {
	c := func(s string) lipgloss.Color { return lipgloss.Color(s) }
	return Stylesheet{
		SelectedPath:     lipgloss.NewStyle().Foreground(c(tc.SelectedPath)),
		FinfoPermissions: lipgloss.NewStyle().Foreground(c(tc.FinfoPermissions)),
		FinfoLastUpdated: lipgloss.NewStyle().Foreground(c(tc.FinfoLastUpdated)),
		FinfoSize:        lipgloss.NewStyle().Foreground(c(tc.FinfoSize)),
		FinfoSep:         lipgloss.NewStyle().Foreground(c(tc.FinfoSep)),
		OperationBar:     lipgloss.NewStyle().Foreground(c(tc.OperationBar)),
		OperationBarInput: lipgloss.NewStyle().Background(c(tc.OperationBarInputBg)),
		ErrBar:  lipgloss.NewStyle().Foreground(c(tc.ErrBar)),
		HelpMsg: lipgloss.NewStyle().Foreground(c(tc.HelpMsg)),
		HelpContent: lipgloss.NewStyle().
			Foreground(c(tc.HelpContent)).
			BorderForeground(c(tc.HelpContent)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderLeft(true),
		TreeRegularFileName: lipgloss.NewStyle().Foreground(c(tc.TreeRegularFile)),
		TreeDirecotryName:   lipgloss.NewStyle().Foreground(c(tc.TreeDirectory)),
		TreeLinkName:        lipgloss.NewStyle().Foreground(c(tc.TreeLink)),
		TreeMarkedNode: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.InnerHalfBlockBorder()).
			Background(c(tc.TreeMarkedBg)),
		TreeSelectionArrow: lipgloss.NewStyle().Foreground(c(tc.TreeSelectionArrow)),
		TreeIndent:         lipgloss.NewStyle().Foreground(c(tc.TreeIndent)),
		TreeIndentSelected: lipgloss.NewStyle().Foreground(c(tc.TreeIndentSelected)),
		PlainTextPreview: lipgloss.NewStyle().
			Italic(true).
			Foreground(c(tc.PlainTextPreview)).
			BorderForeground(c(tc.PlainTextPreviewBorder)).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true),
	}
}

// LoadTheme resolves a theme by name:
//   - "dark" (or empty): built-in dark theme
//   - "light": built-in light theme
//   - anything else: loaded from ~/.config/bt/<name>.yaml (or absolute path)
//
// Unknown/missing custom themes fall back to dark.
func LoadTheme(name string) ThemeColors {
	switch strings.ToLower(name) {
	case "", "dark":
		return DarkThemeColors
	case "light":
		return LightThemeColors
	default:
		path := name
		if !filepath.IsAbs(path) {
			home, err := os.UserHomeDir()
			if err != nil {
				log.Printf("theme: cannot find home dir: %v", err)
				return DarkThemeColors
			}
			path = filepath.Join(home, ".config", "bt", name)
			if filepath.Ext(path) == "" {
				path += ".yaml"
			}
		}
		vp := viper.New()
		vp.SetConfigFile(path)
		if err := vp.ReadInConfig(); err != nil {
			log.Printf("theme: failed to load %q: %v", path, err)
			return DarkThemeColors
		}
		tc := DarkThemeColors // start from dark defaults so partial files work
		if err := vp.Unmarshal(&tc); err != nil {
			log.Printf("theme: failed to parse %q: %v", path, err)
			return DarkThemeColors
		}
		return tc
	}
}
