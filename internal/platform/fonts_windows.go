package platform

import (
	"syscall"
	"unsafe"

	"github.com/pengh/GoLabel/internal/i18n"
	"golang.org/x/sys/windows"
)

const (
	// winFontFactoryShared 使用 DirectWrite 的共享字体工厂
	winFontFactoryShared = 0
	// winFontRelease 释放字体 COM 接口的引用
	winFontRelease = 2
	// winFontSystemCollection 获取系统字体集合
	winFontSystemCollection = 3
	// winFontGetFamily 获取指定序号的字体族
	winFontGetFamily = 4
	// winFontFindFamily 查找原始名称对应的字体族
	winFontFindFamily = 5
	// winFontFamilyNames 获取字体族的多语言名称
	winFontFamilyNames = 6
	// winFontFindLocale 查找指定语言的字体名称
	winFontFindLocale = 4
	// winFontStringLength 获取字体名称的字符数
	winFontStringLength = 7
	// winFontGetString 读取字体名称
	winFontGetString = 8
)

// winFontFactoryID 标识 DirectWrite 字体工厂接口
var winFontFactoryID = windows.GUID{Data1: 0xb859ee5a, Data2: 0xd838, Data3: 0x4b5b, Data4: [8]byte{0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48}}

// winCreateFontFactory 创建 DirectWrite 的字体工厂
var winCreateFontFactory = windows.NewLazySystemDLL("dwrite.dll").NewProc("DWriteCreateFactory")

// winFontObject 保存 DirectWrite 字体 COM 接口的虚函数表指针
type winFontObject struct {
	vtable *[9]uintptr
}

// call 调用字体 COM 接口的方法并返回结果
//
//go:uintptrescapes
func (object *winFontObject) call(method int, args ...uintptr) int32 {
	params := append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)
	result, _, _ := syscall.SyscallN(object.vtable[method], params...)
	return int32(result)
}

// FontFamilyName 在中文界面读取字体的简体中文名称，无中文名称时保留原名
func FontFamilyName(family string) string {
	if i18n.Locale() == "en" {
		return family
	}
	name, err := windows.UTF16PtrFromString(family)
	if err != nil {
		return family
	}
	var factory, collection, font, names *winFontObject
	hr, _, _ := winCreateFontFactory.Call(winFontFactoryShared,
		uintptr(unsafe.Pointer(&winFontFactoryID)), uintptr(unsafe.Pointer(&factory)))
	if int32(hr) < 0 {
		return family
	}
	defer factory.call(winFontRelease)
	if factory.call(winFontSystemCollection, uintptr(unsafe.Pointer(&collection)), 0) < 0 {
		return family
	}
	defer collection.call(winFontRelease)
	var index uint32
	var exists int32
	if collection.call(winFontFindFamily, uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&index)), uintptr(unsafe.Pointer(&exists))) < 0 || exists == 0 {
		return family
	}
	if collection.call(winFontGetFamily, uintptr(index), uintptr(unsafe.Pointer(&font))) < 0 {
		return family
	}
	defer font.call(winFontRelease)
	if font.call(winFontFamilyNames, uintptr(unsafe.Pointer(&names))) < 0 {
		return family
	}
	defer names.call(winFontRelease)
	locale, _ := windows.UTF16PtrFromString("zh-cn")
	if names.call(winFontFindLocale, uintptr(unsafe.Pointer(locale)),
		uintptr(unsafe.Pointer(&index)), uintptr(unsafe.Pointer(&exists))) < 0 || exists == 0 {
		return family
	}
	var length uint32
	if names.call(winFontStringLength, uintptr(index), uintptr(unsafe.Pointer(&length))) < 0 {
		return family
	}
	buffer := make([]uint16, length+1)
	if names.call(winFontGetString, uintptr(index), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer))) < 0 {
		return family
	}
	return windows.UTF16ToString(buffer)
}
