package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
)

// TestMain 为测试提供固定的字体菜单和独立的应用数据目录
func TestMain(m *testing.M) {
	makeLabelFonts = func() fontMenu {
		return fontMenu{
			options:      []string{defaultFontName()},
			nameByFamily: map[string]string{"": defaultFontName()},
			familyByName: map[string]string{defaultFontName(): ""},
		}
	}
	// 无窗口环境不查询系统外观，默认按浅色处理
	systemDark = func() bool { return false }
	// 无窗口环境不读系统语言，统一按简体中文处理
	systemLanguage = func() string { return string(languageZhHans) }
	// 主题缩略图和配色在测试里都要可预期
	applyTheme()
	dir, err := os.MkdirTemp("", "golabel-test")
	if err != nil {
		panic(err)
	}
	settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	mygo.App.SetPath(mygo.PathUserData, dir)
	prefs = defaultPreferences()
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
