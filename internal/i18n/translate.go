package i18n

import (
	"fmt"

	"github.com/egoist/mygo"
)

// Locale 读取 MyGo 已解析的界面语言，可从后台读取
var Locale = mygo.App.Locale

// Text 是一条文字在两种语言下的写法
type Text struct{ ZH, EN string }

// Get 返回这条文字在当前语言下的写法
func (t Text) Get() string {
	if Locale() == "en" {
		return t.EN
	}
	return t.ZH
}

// Tr 返回 key 在当前语言下的文字；缺少词条时返回 key 本身，便于发现遗漏
func Tr(key string) string {
	message, ok := messages[key]
	if !ok {
		return key
	}
	return message.Get()
}

// Trf 返回把参数代入占位符后的界面文字
func Trf(key string, args ...any) string {
	return fmt.Sprintf(Tr(key), args...)
}

// Lookup 返回文案的中英文版本，供模板搜索使用
func Lookup(key string) (Text, bool) {
	value, ok := messages[key]
	return value, ok
}
