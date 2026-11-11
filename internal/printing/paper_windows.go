package printing

import (
	"errors"
	"math"
	"unsafe"

	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"golang.org/x/sys/windows"
)

const (
	// winPaperFields 标记方向、纸型、纸长、纸宽和缩放字段有效
	winPaperFields = 0x00000001 | 0x00000002 | 0x00000004 | 0x00000008 | 0x00000010
	// winFormNameField 标记驱动使用命名纸型
	winFormNameField = 0x00010000
	// winMergeDevMode 请求驱动合并并返回打印设置
	winMergeDevMode = 0x00000008 | 0x00000002
	// winPortrait 保持纸宽沿打印头方向、纸高沿进纸方向
	winPortrait = 1
	// winPaperToleranceMM 允许驱动以十分之一毫米和整数打印点量化纸张尺寸
	winPaperToleranceMM = 0.2
)

// winDevMode 对应 DEVMODEW 中截至打印质量字段的公共前缀
type winDevMode struct {
	deviceName                                      [32]uint16
	specVersion, driverVersion, size, driverExtra   uint16
	fields                                          uint32
	orientation, paperSize, paperLength, paperWidth int16
	scale, copies, defaultSource, printQuality      int16
}

// winDevNames 对应系统打印对话框返回的 DEVNAMES 字符串偏移
type winDevNames struct {
	driverOffset, deviceOffset, outputOffset, defaultPrinter uint16
}

// setPage 将标签尺寸和原尺寸缩放写入驱动设置并保留其他打印选项
func (mode *winDevMode) setPage(page platform.Page) {
	mode.fields = mode.fields&^winFormNameField | winPaperFields
	mode.orientation, mode.paperSize = winPortrait, 0
	mode.paperWidth = int16(math.Round(page.WidthMM * 10))
	mode.paperLength = int16(math.Round(page.HeightMM * 10))
	mode.scale = 100
}

// configureWindowsPage 将标签纸型合并到所选驱动并更新打印机设备上下文
func configureWindowsPage(dlg *winPrintDialog, page platform.Page) error {
	failure := errors.New(i18n.Tr("error.printSettings"))
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	lock, unlock := kernel.NewProc("GlobalLock"), kernel.NewProc("GlobalUnlock")
	modeAddress, _, _ := lock.Call(dlg.devMode)
	if modeAddress == 0 {
		return failure
	}
	defer unlock.Call(dlg.devMode)
	namesAddress, _, _ := lock.Call(dlg.devNames)
	if namesAddress == 0 {
		return failure
	}
	defer unlock.Call(dlg.devNames)
	names := (*winDevNames)(unsafe.Pointer(namesAddress))
	device := namesAddress + uintptr(names.deviceOffset)*unsafe.Sizeof(uint16(0))
	spool := windows.NewLazySystemDLL("winspool.drv")
	var printer windows.Handle
	if result, _, _ := spool.NewProc("OpenPrinterW").Call(device, uintptr(unsafe.Pointer(&printer)), 0); result == 0 {
		return failure
	}
	defer spool.NewProc("ClosePrinter").Call(uintptr(printer))

	mode := (*winDevMode)(unsafe.Pointer(modeAddress))
	mode.setPage(page)
	result, _, _ := spool.NewProc("DocumentPropertiesW").Call(
		dlg.owner, uintptr(printer), device, modeAddress, modeAddress, winMergeDevMode,
	)
	if int32(result) != 1 {
		return failure
	}
	if result, _, _ := winGDI.NewProc("ResetDCW").Call(dlg.dc, modeAddress); result == 0 {
		return failure
	}
	width := float64(windowsPrintCapability(dlg.dc, winPhysicalWidth)) * 25.4 / float64(windowsPrintCapability(dlg.dc, winDPIx))
	height := float64(windowsPrintCapability(dlg.dc, winPhysicalHeight)) * 25.4 / float64(windowsPrintCapability(dlg.dc, winDPIy))
	if math.Abs(width-page.WidthMM) > winPaperToleranceMM || math.Abs(height-page.HeightMM) > winPaperToleranceMM {
		return errors.New(i18n.Trf("error.printPaperSize", width, height, page.WidthMM, page.HeightMM))
	}
	return nil
}
