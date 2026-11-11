//go:build darwin

package main

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// detectSystemLanguage 读取系统首选语言列表的第一项，如 "zh-Hans-CN" 或 "en-US"
func detectSystemLanguage() string {
	if !loadFoundation() {
		return ""
	}
	defer macPool()()
	defaults := macSend[objc.ID](macClass("NSUserDefaults"), "standardUserDefaults")
	languages := macSend[objc.ID](defaults, "arrayForKey:", macString("AppleLanguages"))
	if languages == 0 {
		return ""
	}
	first := macSend[objc.ID](languages, "firstObject")
	if first == 0 {
		return ""
	}
	text := macSend[*byte](first, "UTF8String")
	if text == nil {
		return ""
	}
	// NSUTF8StringEncoding 为 4
	return unsafe.String(text, macSend[int](first, "lengthOfBytesUsingEncoding:", 4))
}
