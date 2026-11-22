package printing

import (
	"errors"
	"math"
	"slices"
	"unsafe"

	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"golang.org/x/sys/windows"
)

const (
	// winPaperFields 标记方向、纸型和缩放字段有效
	winPaperFields = 0x00000001 | 0x00000002 | 0x00000010
	// winPaperDimensionsFields 标记自定义纸长和纸宽字段有效
	winPaperDimensionsFields = 0x00000004 | 0x00000008
	// winFormNameField 标记驱动使用命名纸型
	winFormNameField = 0x00010000
	// winPaperIDsCapability 查询驱动支持的纸型编号
	winPaperIDsCapability = 2
	// winPaperSizesCapability 查询驱动支持的纸型尺寸
	winPaperSizesCapability = 3
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

// winPaperDimensions 对应驱动以十分之一毫米返回的纸宽和纸长
type winPaperDimensions struct {
	width, length int32
}

// setPage 设置匹配的纸型或自定义尺寸，并保留其余打印选项
func (mode *winDevMode) setPage(size winPaperDimensions, paperID uint16) {
	mode.fields = mode.fields&^(winFormNameField|winPaperDimensionsFields) | winPaperFields
	mode.orientation, mode.paperSize = winPortrait, int16(paperID)
	if paperID == 0 {
		mode.fields |= winPaperDimensionsFields
		mode.paperWidth, mode.paperLength = int16(size.width), int16(size.length)
	}
	mode.scale = 100
}

// matchingWindowsPaper 查找驱动中尺寸相同的纸型编号
func matchingWindowsPaper(capabilities *windows.LazyProc, device, port, mode uintptr, size winPaperDimensions) uint16 {
	count, _, _ := capabilities.Call(device, port, winPaperSizesCapability, 0, mode)
	idCount, _, _ := capabilities.Call(device, port, winPaperIDsCapability, 0, mode)
	if int32(count) <= 0 || count != idCount {
		return 0
	}
	sizes := make([]winPaperDimensions, count)
	n, _, _ := capabilities.Call(device, port, winPaperSizesCapability, uintptr(unsafe.Pointer(&sizes[0])), mode)
	if n != count {
		return 0
	}
	index := slices.Index(sizes, size)
	if index < 0 {
		return 0
	}
	ids := make([]uint16, count)
	n, _, _ = capabilities.Call(device, port, winPaperIDsCapability, uintptr(unsafe.Pointer(&ids[0])), mode)
	if n != count {
		return 0
	}
	return ids[index]
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
	port := namesAddress + uintptr(names.outputOffset)*unsafe.Sizeof(uint16(0))
	spool := windows.NewLazySystemDLL("winspool.drv")
	var printer windows.Handle
	if result, _, _ := spool.NewProc("OpenPrinterW").Call(device, uintptr(unsafe.Pointer(&printer)), 0); result == 0 {
		return failure
	}
	defer spool.NewProc("ClosePrinter").Call(uintptr(printer))

	mode := (*winDevMode)(unsafe.Pointer(modeAddress))
	size := winPaperDimensions{int32(math.Round(page.WidthMM * 10)), int32(math.Round(page.HeightMM * 10))}
	paperID := matchingWindowsPaper(spool.NewProc("DeviceCapabilitiesW"), device, port, modeAddress, size)
	mode.setPage(size, paperID)
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
