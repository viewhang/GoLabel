//go:build windows

package main

import "golang.org/x/sys/windows/registry"

// detectSystemLanguage 读取用户的首选界面语言，如 "zh-CN" 或 "en-US"；
// 系统没有记录首选语言时退回区域格式的语言名称
func detectSystemLanguage() string {
	if languages := preferredUILanguages(); len(languages) > 0 {
		return languages[0]
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\International`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	name, _, err := key.GetStringValue("LocaleName")
	if err != nil {
		return ""
	}
	return name
}

// preferredUILanguages 返回 Control Panel\Desktop 里记下的首选界面语言列表
func preferredUILanguages() []string {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer key.Close()
	languages, _, err := key.GetStringsValue("PreferredUILanguages")
	if err != nil {
		return nil
	}
	return languages
}
