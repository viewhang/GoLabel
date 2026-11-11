//go:build !darwin && !windows

package main

import (
	"os"
	"strings"
)

// detectSystemLanguage 按环境变量里的界面语言判断，LANGUAGE 优先，其次是 LC_ALL 和 LANG
func detectSystemLanguage() string {
	for _, name := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		// LANGUAGE 可能是冒号分隔的语言列表，取第一项
		if i := strings.IndexByte(value, ':'); i >= 0 {
			value = value[:i]
		}
		return value
	}
	return ""
}
