//go:build !darwin && !windows

package platform

import (
	"os"
	"strings"
)

// SystemDark 在 Linux 和 BSD 上按桌面环境的环境变量判断深色外观
func SystemDark() bool {
	if strings.Contains(strings.ToLower(os.Getenv("GTK_THEME")), "dark") {
		return true
	}
	return strings.Contains(strings.ToLower(os.Getenv("COLOR_SCHEME")), "dark")
}
