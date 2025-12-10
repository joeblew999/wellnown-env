// Package viatheme provides shared theme utilities for Via web dashboards.
//
// This package consolidates the picocss theme mapping used across Via examples.
//
// Usage:
//
//	import "github.com/joeblew999/wellnown-env/pkg/viatheme"
//
//	theme, name := viatheme.GetFromEnv("purple") // default to purple
//	v := via.New(picocss.Plugin(theme))
package viatheme

import (
	"os"
	"strings"

	"github.com/go-via/via-plugin-picocss/picocss"
)

// ThemeMap maps theme names to picocss.Theme constants.
var ThemeMap = map[string]picocss.Theme{
	"amber":   picocss.ThemeAmber,
	"blue":    picocss.ThemeBlue,
	"cyan":    picocss.ThemeCyan,
	"fuchsia": picocss.ThemeFuchia,
	"green":   picocss.ThemeGreen,
	"grey":    picocss.ThemeGrey,
	"indigo":  picocss.ThemeIndigo,
	"jade":    picocss.ThemeJade,
	"lime":    picocss.ThemeLime,
	"orange":  picocss.ThemeOrange,
	"pink":    picocss.ThemePink,
	"pumpkin": picocss.ThemePumpkin,
	"purple":  picocss.ThemePurple,
	"red":     picocss.ThemeRed,
	"sand":    picocss.ThemeSand,
	"slate":   picocss.ThemeSlate,
	"violet":  picocss.ThemeViolet,
	"yellow":  picocss.ThemeYellow,
	"zinc":    picocss.ThemeZinc,
}

// ThemeNames is a sorted list of available theme names.
var ThemeNames = []string{
	"amber", "blue", "cyan", "fuchsia", "green",
	"grey", "indigo", "jade", "lime", "orange",
	"pink", "pumpkin", "purple", "red", "sand",
	"slate", "violet", "yellow", "zinc",
}

// Get returns the picocss.Theme for a theme name.
// Returns the default theme if not found.
func Get(name string, defaultTheme picocss.Theme) picocss.Theme {
	name = strings.ToLower(name)
	if theme, ok := ThemeMap[name]; ok {
		return theme
	}
	return defaultTheme
}

// GetFromEnv reads VIA_THEME env var and returns the corresponding theme.
// The defaultName is used if VIA_THEME is not set or invalid.
// Returns both the theme constant and the resolved name.
func GetFromEnv(defaultName string) (picocss.Theme, string) {
	themeName := strings.ToLower(os.Getenv("VIA_THEME"))
	if themeName == "" {
		themeName = defaultName
	}
	if theme, ok := ThemeMap[themeName]; ok {
		return theme, themeName
	}
	// Fallback to default
	if theme, ok := ThemeMap[defaultName]; ok {
		return theme, defaultName
	}
	return picocss.ThemePurple, "purple"
}
