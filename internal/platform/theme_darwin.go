//go:build darwin

package platform

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/pengh/GoLabel/internal/platform/cocoa"
)

// foundationRuntime 保证读取系统外观前已经加载 Foundation
var foundationRuntime struct {
	once sync.Once
	ok   bool
}

// loadFoundation 加载读取系统设置所需的框架
func loadFoundation() bool {
	foundationRuntime.once.Do(func() {
		for _, path := range []string{
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			"/usr/lib/libobjc.A.dylib",
		} {
			if _, err := purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL); err != nil {
				return
			}
		}
		foundationRuntime.ok = true
	})
	return foundationRuntime.ok
}

// SystemDark 读取系统的外观设置：深色时 AppleInterfaceStyle 为 "Dark"
func SystemDark() bool {
	if !loadFoundation() {
		return false
	}
	defer cocoa.Pool()()
	defaults := cocoa.Send[objc.ID](cocoa.Class("NSUserDefaults"), "standardUserDefaults")
	style := cocoa.Send[objc.ID](defaults, "stringForKey:", cocoa.String("AppleInterfaceStyle"))
	if style == 0 {
		return false
	}
	text := cocoa.Send[*byte](style, "UTF8String")
	if text == nil {
		return false
	}
	// NSUTF8StringEncoding 为 4
	length := cocoa.Send[int](style, "lengthOfBytesUsingEncoding:", 4)
	return unsafe.String(text, length) == "Dark"
}
