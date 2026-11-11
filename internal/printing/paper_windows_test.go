package printing

import (
	"testing"
	"unsafe"

	"github.com/pengh/GoLabel/internal/platform"
)

// TestWindowsDevModeLayout 验证纸张设置字段与 Win32 DEVMODEW 的内存偏移一致
func TestWindowsDevModeLayout(t *testing.T) {
	var mode winDevMode
	for _, field := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"size", unsafe.Offsetof(mode.size), 68},
		{"fields", unsafe.Offsetof(mode.fields), 72},
		{"orientation", unsafe.Offsetof(mode.orientation), 76},
		{"paperSize", unsafe.Offsetof(mode.paperSize), 78},
		{"paperLength", unsafe.Offsetof(mode.paperLength), 80},
		{"paperWidth", unsafe.Offsetof(mode.paperWidth), 82},
		{"scale", unsafe.Offsetof(mode.scale), 84},
		{"prefix", unsafe.Sizeof(mode), 92},
	} {
		if field.got != field.want {
			t.Errorf("%s 的偏移或大小为 %d，期望 %d", field.name, field.got, field.want)
		}
	}
	if unsafe.Sizeof(winDevNames{}) != 8 {
		t.Fatal("DEVNAMES 的字段布局有误")
	}
}

// TestWindowsPageSettings 验证标签尺寸覆盖旧纸型和缩放且保留份数及驱动参数
func TestWindowsPageSettings(t *testing.T) {
	for _, test := range []struct {
		page                  platform.Page
		wantWidth, wantLength int16
	}{
		{platform.Page{WidthMM: 40, HeightMM: 30}, 400, 300},
		{platform.Page{WidthMM: 30, HeightMM: 40}, 300, 400},
		{platform.Page{WidthMM: 40.14, HeightMM: 30.16}, 401, 302},
	} {
		mode := winDevMode{
			size: 220, driverExtra: 32, fields: winFormNameField | 0x00000100,
			orientation: 2, paperSize: 9, paperWidth: 2100, paperLength: 2970,
			scale: 50, copies: 3, defaultSource: 7, printQuality: 203,
		}
		mode.setPage(test.page)
		if mode.orientation != winPortrait || mode.paperSize != 0 || mode.scale != 100 {
			t.Fatal("打印设置未覆盖旧方向、纸型或缩放")
		}
		if mode.paperWidth != test.wantWidth || mode.paperLength != test.wantLength {
			t.Fatalf("纸张宽高为 %d × %d，期望 %d × %d", mode.paperWidth, mode.paperLength, test.wantWidth, test.wantLength)
		}
		if mode.fields != winPaperFields|0x00000100 {
			t.Fatal("未清除旧纸型名称或丢失了份数字段标记")
		}
		if mode.size != 220 || mode.driverExtra != 32 || mode.copies != 3 || mode.defaultSource != 7 || mode.printQuality != 203 {
			t.Fatal("修改纸型时覆盖了其他驱动参数")
		}
	}
}
