//go:build windows

package main

import "golang.org/x/sys/windows/registry"

// detectSystemDark 读取 Windows 的应用主题偏好，AppsUseLightTheme 为 0 表示深色
func detectSystemDark() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false
	}
	return value == 0
}
