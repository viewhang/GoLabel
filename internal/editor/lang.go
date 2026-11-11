package editor

import (
	"github.com/egoist/mygo"
	"github.com/pengh/GoLabel/internal/i18n"
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

// setAppLocale 设置应用界面和原生控件共用的语言
var setAppLocale = mygo.App.SetLocale

// languageTitle 返回语言在设置里显示的名称
func languageTitle(l language) string {
	switch l {
	case languageZhHans:
		// 语言名称用该语言自己的写法，不看当前界面也能找到
		return "简体中文"
	case languageEnglish:
		return "English"
	default:
		return i18n.Tr("settings.system")
	}
}

// applyLanguage 将语言偏好交给 MyGo，空值表示跟随系统
func applyLanguage() {
	switch language(prefs.Language) {
	case languageZhHans, languageEnglish:
		setAppLocale(prefs.Language)
	default:
		setAppLocale("")
	}
}

// useLanguage 保存语言偏好并立即生效
func useLanguage(l language) {
	prefs.Language = string(l)
	applyLanguage()
	savePreferences()
}
