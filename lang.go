package main

import (
	"fmt"
	"strings"
)

// language 既是界面显示的语言，也是设置里的三个选项
type language string

const (
	// languageSystem 表示跟随系统首选语言
	languageSystem language = "system"
	// languageZhHans 是简体中文，也是应用自己的语言
	languageZhHans language = "zh-Hans"
	// languageEnglish 是英文
	languageEnglish language = "en"
)

// languageOptions 是设置里的三个选项，顺序即显示顺序
var languageOptions = [...]language{languageZhHans, languageEnglish, languageSystem}

// currentLang 是当前界面语言，界面文字都通过 tr 读取它
var currentLang = languageZhHans

// systemLanguage 查询系统首选语言，由各平台的实现提供；测试中会替换掉
var systemLanguage = detectSystemLanguage

// appLocale 把界面语言告诉原生界面，让菜单和系统对话框跟上；main 启动时接入
var appLocale = func(string) {}

// text 是一条文字在两种语言下的写法
type text struct{ zh, en string }

// get 返回这条文字在当前语言下的写法
func (t text) get() string {
	if currentLang == languageEnglish {
		return t.en
	}
	return t.zh
}

// tr 返回 key 在当前语言下的文字；缺少词条时返回 key 本身，便于发现遗漏
func tr(key string) string {
	message, ok := messages[key]
	if !ok {
		return key
	}
	return message.get()
}

// trf 返回把参数代入占位符后的界面文字
func trf(key string, args ...any) string {
	return fmt.Sprintf(tr(key), args...)
}

// languageTitle 返回语言在设置里显示的名称
func languageTitle(l language) string {
	switch l {
	case languageZhHans:
		// 语言名称用该语言自己的写法，不看当前界面也能找到
		return "简体中文"
	case languageEnglish:
		return "English"
	default:
		return tr("settings.system")
	}
}

// languageStatus 返回切换语言后显示在顶栏的提示，用切换后的语言写
func languageStatus(l language) string {
	switch l {
	case languageZhHans:
		return tr("settings.switchedZh")
	case languageEnglish:
		return tr("settings.switchedEn")
	default:
		return tr("settings.switchedSystem")
	}
}

// applyLanguage 按偏好确定界面语言，「跟随系统」时读取系统首选语言
func applyLanguage() language {
	switch language(prefs.Language) {
	case languageZhHans:
		currentLang = languageZhHans
	case languageEnglish:
		currentLang = languageEnglish
	default:
		currentLang = matchLanguage(systemLanguage())
	}
	return currentLang
}

// matchLanguage 把系统语言标签折算成界面支持的语言：中文一律用简体，其他语言用英文
func matchLanguage(tag string) language {
	tag = strings.ToLower(strings.TrimSpace(tag))
	switch {
	case tag == "":
		// 读不到系统语言时用应用自己的语言
		return languageZhHans
	case strings.HasPrefix(tag, "zh"):
		// 界面只提供一套中文，繁体系统也用简体
		return languageZhHans
	default:
		return languageEnglish
	}
}

// useLanguage 保存语言偏好并立即生效
func useLanguage(l language) {
	prefs.Language = string(l)
	applyLanguage()
	// 字体菜单里的「系统默认」等名称随语言变化，切换后重新枚举
	resetFontMenu()
	savePreferences()
	appLocale(prefs.Language)
}
