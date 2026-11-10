package main

import (
	"fmt"
	"image"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// winPrintFlags 请求打印机 DC 和驱动份数设置，并隐藏不适用的选项
	winPrintFlags = 0x00000100 | 0x00040000 | 0x00100000 | 0x00800000 | 0x00000008 | 0x00000004
	// winPrintGeneral 是系统打印对话框的常规页面
	winPrintGeneral = ^uint32(0)
	// winPrintConfirmed 表示用户确认打印
	winPrintConfirmed = 1
	// winColorOnColor 使用原像素色值缩放标签位图
	winColorOnColor = 3
	// winSourceCopy 将源位图像素复制到打印机
	winSourceCopy = 0x00CC0020
)

// winGDI 提供 Windows 打印机和位图绘制接口
var winGDI = windows.NewLazySystemDLL("gdi32.dll")

// winPrintDialog 对应 Win32 PRINTDLGEXW 的字段布局
type winPrintDialog struct {
	size                         uint32
	owner, devMode, devNames, dc uintptr
	flags, flags2, exclusions    uint32
	pageRanges, maxPageRanges    uint32
	ranges                       uintptr
	minPage, maxPage, copies     uint32
	instance, template, callback uintptr
	propertyPages                uint32
	properties                   uintptr
	startPage, action            uint32
}

// winDocInfo 对应 Win32 DOCINFOW 的字段布局
type winDocInfo struct {
	size                   int32
	name, output, dataType *uint16
	flags                  uint32
}

// winBitmapInfo 对应不带调色板的 Win32 BITMAPINFOHEADER
type winBitmapInfo struct {
	size                             uint32
	width, height                    int32
	planes, bitCount                 uint16
	compression, imageSize           uint32
	xPixelsPerMeter, yPixelsPerMeter int32
	colorsUsed, colorsImportant      uint32
}

// print 打开 Windows 系统打印对话框并提交用户确认的标签
func (g *labelGraphics) print(l *label, title string, owner uintptr) (bool, error) {
	raster, err := l.render()
	if err != nil {
		return false, err
	}
	dlg := winPrintDialog{
		size: uint32(unsafe.Sizeof(winPrintDialog{})), owner: owner, flags: winPrintFlags,
		minPage: 1, maxPage: 1, copies: 1, startPage: winPrintGeneral,
	}
	printDlg := windows.NewLazySystemDLL("comdlg32.dll").NewProc("PrintDlgExW")
	result, _, _ := printDlg.Call(uintptr(unsafe.Pointer(&dlg)))
	free := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalFree")
	if dlg.devMode != 0 {
		defer free.Call(dlg.devMode)
	}
	if dlg.devNames != 0 {
		defer free.Call(dlg.devNames)
	}
	if dlg.dc != 0 {
		defer winGDI.NewProc("DeleteDC").Call(dlg.dc)
	}
	if int32(result) < 0 {
		return false, fmt.Errorf("无法打开系统打印对话框（0x%08X）", uint32(result))
	}
	if dlg.action != winPrintConfirmed {
		return false, nil
	}
	if err := sendWindowsPrintJob(dlg.dc, title, l, raster); err != nil {
		return false, err
	}
	return true, nil
}

// sendWindowsPrintJob 将标签按打印机的实际点距提交为一页
func sendWindowsPrintJob(dc uintptr, title string, l *label, raster *image.RGBA) error {
	name, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	info := winDocInfo{size: int32(unsafe.Sizeof(winDocInfo{})), name: name}
	if result, _, _ := winGDI.NewProc("StartDocW").Call(dc, uintptr(unsafe.Pointer(&info))); int32(result) <= 0 {
		return fmt.Errorf("无法创建打印任务")
	}
	completed := false
	defer func() {
		if !completed {
			winGDI.NewProc("AbortDoc").Call(dc)
		}
	}()
	if result, _, _ := winGDI.NewProc("StartPage").Call(dc); int32(result) <= 0 {
		return fmt.Errorf("无法开始打印页面")
	}
	if err := drawWindowsLabel(dc, l, raster); err != nil {
		return err
	}
	if result, _, _ := winGDI.NewProc("EndPage").Call(dc); int32(result) <= 0 {
		return fmt.Errorf("无法完成打印页面")
	}
	if result, _, _ := winGDI.NewProc("EndDoc").Call(dc); int32(result) <= 0 {
		return fmt.Errorf("无法提交打印任务")
	}
	completed = true
	return nil
}

// drawWindowsLabel 将 RGBA 位图转换为 Windows DIB 并按纸张原点绘制
func drawWindowsLabel(dc uintptr, l *label, raster *image.RGBA) error {
	width, height := raster.Rect.Dx(), raster.Rect.Dy()
	bitmap := winBitmapInfo{size: uint32(unsafe.Sizeof(winBitmapInfo{})), width: int32(width), height: -int32(height), planes: 1, bitCount: 32}
	for i := 0; i < len(raster.Pix); i += 4 {
		raster.Pix[i], raster.Pix[i+2] = raster.Pix[i+2], raster.Pix[i]
	}

	getCaps := winGDI.NewProc("GetDeviceCaps")
	capability := func(index uintptr) int32 {
		value, _, _ := getCaps.Call(dc, index)
		return int32(value)
	}
	const (
		// winDPIx 和 winDPIy 是打印机的横向和纵向 DPI 查询项
		winDPIx, winDPIy = 88, 90
		// winOffsetX 和 winOffsetY 是可打印区域相对纸张边缘的偏移查询项
		winOffsetX, winOffsetY = 112, 113
	)
	destW := uintptr(math.Round(l.Width * float64(capability(winDPIx)) / 25.4))
	destH := uintptr(math.Round(l.Height * float64(capability(winDPIy)) / 25.4))
	winGDI.NewProc("SetStretchBltMode").Call(dc, winColorOnColor)
	rows, _, _ := winGDI.NewProc("StretchDIBits").Call(
		dc, uintptr(-capability(winOffsetX)), uintptr(-capability(winOffsetY)), destW, destH,
		0, 0, uintptr(width), uintptr(height), uintptr(unsafe.Pointer(unsafe.SliceData(raster.Pix))),
		uintptr(unsafe.Pointer(&bitmap)), 0, winSourceCopy,
	)
	if int32(rows) <= 0 {
		return fmt.Errorf("无法绘制打印标签，请检查打印机纸型")
	}
	return nil
}
