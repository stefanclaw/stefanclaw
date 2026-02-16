package config

import "testing"

func TestDetectLanguage_FromLANG(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "de_DE.UTF-8")
	t.Setenv("LANGUAGE", "")

	got := DetectLanguage()
	if got != "Deutsch" {
		t.Errorf("DetectLanguage() = %q, want Deutsch", got)
	}
}

func TestDetectLanguage_FromLC_ALL(t *testing.T) {
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LANGUAGE", "")

	got := DetectLanguage()
	if got != "Français" {
		t.Errorf("DetectLanguage() = %q, want Français (LC_ALL takes priority)", got)
	}
}

func TestDetectLanguage_FromLANGUAGE(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "")
	t.Setenv("LANGUAGE", "es_ES")

	got := DetectLanguage()
	if got != "Español" {
		t.Errorf("DetectLanguage() = %q, want Español", got)
	}
}

func TestDetectLanguage_Unset(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "")
	t.Setenv("LANGUAGE", "")

	got := DetectLanguage()
	if got != "English" {
		t.Errorf("DetectLanguage() = %q, want English (fallback)", got)
	}
}

func TestDetectLanguage_UnrecognizedLocale(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LANG", "xx_XX.UTF-8")
	t.Setenv("LANGUAGE", "")

	got := DetectLanguage()
	if got != "English" {
		t.Errorf("DetectLanguage() = %q, want English (unrecognized)", got)
	}
}

func TestParseLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   string
	}{
		{"de_DE.UTF-8", "Deutsch"},
		{"en_US.UTF-8", "English"},
		{"es_ES", "Español"},
		{"fr_FR.UTF-8", "Français"},
		{"ja_JP.UTF-8", "日本語"},
		{"zh_CN", "中文"},
		{"pt_BR.UTF-8", "Português"},
		{"it_IT", "Italiano"},
		{"ko_KR.UTF-8", "한국어"},
		{"ru_RU.UTF-8", "Русский"},
		{"C", "English"},
		{"POSIX", "English"},
		{"de", "Deutsch"},
		{"de-DE", "Deutsch"},
	}

	for _, tt := range tests {
		got := parseLocale(tt.locale)
		if got != tt.want {
			t.Errorf("parseLocale(%q) = %q, want %q", tt.locale, got, tt.want)
		}
	}
}
