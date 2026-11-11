package cocoa

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// Send 调用 Objective-C 对象的方法
func Send[T any](object objc.ID, method string, args ...any) T {
	return objc.Send[T](object, objc.RegisterName(method), args...)
}

// Class 查找系统框架中的 Objective-C 类
func Class(name string) objc.ID {
	return objc.ID(objc.GetClass(name))
}

// String 构造供 Objective-C 接口读取的 UTF-8 字符串
func String(value string) objc.ID {
	return Send[objc.ID](Class("NSString"), "stringWithUTF8String:", value)
}

// Data 将 Go 字节复制为 NSData
func Data(data []byte) objc.ID {
	result := Send[objc.ID](Class("NSData"), "dataWithBytes:length:", unsafe.SliceData(data), len(data))
	runtime.KeepAlive(data)
	return result
}

// Pool 创建作用域内的 Objective-C 自动释放池
func Pool() func() {
	pool := Send[objc.ID](Class("NSAutoreleasePool"), "new")
	return func() { Send[objc.ID](pool, "drain") }
}
