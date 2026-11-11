package main

import "time"

// themeMode 是外观设置里的三种选择
type themeMode int

const (
	// themeModeLight 固定使用浅色
	themeModeLight themeMode = iota
	// themeModeDark 固定使用深色
	themeModeDark
	// themeModeSystem 跟随系统外观
	themeModeSystem
)

// 外观模式在配置文件中保存的名称
const (
	themeNameLight  = "light"
	themeNameDark   = "dark"
	themeNameSystem = "system"
)

// themeModeName 返回外观模式保存到配置文件时的名称
func themeModeName(m themeMode) string {
	switch m {
	case themeModeLight:
		return themeNameLight
	case themeModeDark:
		return themeNameDark
	default:
		return themeNameSystem
	}
}

// parseThemeMode 解析配置中的外观模式，无法识别时按跟随系统处理
func parseThemeMode(name string) themeMode {
	switch name {
	case themeNameLight:
		return themeModeLight
	case themeNameDark:
		return themeModeDark
	default:
		return themeModeSystem
	}
}

// systemDark 查询系统外观，由各平台的实现提供；测试中会替换掉
var systemDark = detectSystemDark

// systemDarkCache 缓存系统外观，避免每帧都去查询系统设置
var systemDarkCache struct {
	at   time.Time
	dark bool
}

// systemAppearanceDark 返回系统当前是否为深色，一秒内复用上次结果
func systemAppearanceDark() bool {
	now := time.Now()
	if now.Sub(systemDarkCache.at) < time.Second {
		return systemDarkCache.dark
	}
	systemDarkCache.at, systemDarkCache.dark = now, systemDark()
	return systemDarkCache.dark
}

// applyTheme 按偏好切换当前配色，返回是否使用了深色
func applyTheme() bool {
	dark := false
	switch parseThemeMode(prefs.Theme) {
	case themeModeDark:
		dark = true
	case themeModeSystem:
		dark = systemAppearanceDark()
	}
	if dark {
		colors = darkPalette
	} else {
		colors = lightPalette
	}
	return dark
}

// resetSystemDarkCache 让下一次查询重新读取系统外观
func resetSystemDarkCache() {
	systemDarkCache.at = time.Time{}
}

// useThemeMode 保存外观模式并立即生效，「跟随系统」会重新查询系统外观
func useThemeMode(m themeMode) {
	prefs.Theme = themeModeName(m)
	resetSystemDarkCache()
	applyTheme()
	savePreferences()
}

// themeModeTitle 返回外观模式在设置页和提示里使用的名称
func themeModeTitle(m themeMode) string {
	switch m {
	case themeModeLight:
		return tr("settings.light")
	case themeModeDark:
		return tr("settings.dark")
	default:
		return tr("settings.system")
	}
}
