package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain 在测试中使用固定的字体菜单和独立的设置文件，避免访问用户环境
func TestMain(m *testing.M) {
	makeLabelFonts = func() fontMenu {
		return fontMenu{
			options:      []string{defaultFontName},
			nameByFamily: map[string]string{"": defaultFontName},
			familyByName: map[string]string{defaultFontName: ""},
		}
	}
	dir, err := os.MkdirTemp("", "golabel-test")
	if err == nil {
		settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	}
	prefs = defaultPreferences()
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}
