package config

import (
	"os"
	"strings"
)

// localeToLanguage maps locale prefixes to human-readable language names.
var localeToLanguage = map[string]string{
	"de": "Deutsch",
	"en": "English",
	"es": "Español",
	"fr": "Français",
	"it": "Italiano",
	"ja": "日本語",
	"ko": "한국어",
	"nl": "Nederlands",
	"pl": "Polski",
	"pt": "Português",
	"ru": "Русский",
	"tr": "Türkçe",
	"uk": "Українська",
	"zh": "中文",
}

// DetectLanguage reads the system locale from environment variables and returns
// a human-readable language name. Falls back to "English" if unset or unrecognized.
func DetectLanguage() string {
	locale := ""
	for _, env := range []string{"LC_ALL", "LANG", "LANGUAGE"} {
		if v := os.Getenv(env); v != "" {
			locale = v
			break
		}
	}
	if locale == "" {
		return "English"
	}
	return parseLocale(locale)
}

// parseLocale extracts the language prefix from a locale string like "de_DE.UTF-8"
// and returns the human-readable language name.
func parseLocale(locale string) string {
	// Strip encoding (e.g., ".UTF-8")
	if idx := strings.Index(locale, "."); idx != -1 {
		locale = locale[:idx]
	}
	// Strip region (e.g., "_DE")
	if idx := strings.Index(locale, "_"); idx != -1 {
		locale = locale[:idx]
	}
	// Also handle hyphen variants (e.g., "de-DE")
	if idx := strings.Index(locale, "-"); idx != -1 {
		locale = locale[:idx]
	}

	locale = strings.ToLower(strings.TrimSpace(locale))

	if name, ok := localeToLanguage[locale]; ok {
		return name
	}
	return "English"
}
