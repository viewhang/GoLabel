package main

import (
	"bytes"
	"cmp"
	"errors"
	"image"
	"image/png"
	"runtime"
	"structs"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// cgSize 表示 Apple 图形接口使用的尺寸
type cgSize struct {
	_    structs.HostLayout
	W, H float64
}

// cgRect 表示 Apple 图形接口使用的矩形
type cgRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// labelGraphics 持有条码生成器和 PDF 绘制接口
type labelGraphics struct {
	ci objc.ID

	createImage func(objc.ID, objc.SEL, objc.ID, cgRect) uintptr
	setSize     func(objc.ID, objc.SEL, cgSize)
	setMargin   func(objc.ID, objc.SEL, float64)

	newConsumer      func(objc.ID) uintptr
	releaseConsumer  func(uintptr)
	newPDF           func(uintptr, *cgRect, uintptr) uintptr
	beginPage        func(uintptr, uintptr)
	drawImage        func(uintptr, cgRect, uintptr)
	setInterpolation func(uintptr, int32)
	endPage          func(uintptr)
	closePDF         func(uintptr)
	releaseContext   func(uintptr)
	releaseImage     func(uintptr)
}

// openGraphics 加载标签渲染和系统打印所需的 Apple 框架
func openGraphics() (*labelGraphics, error) {
	for _, name := range []string{"AppKit", "CoreImage", "PDFKit"} {
		if _, err := purego.Dlopen("/System/Library/Frameworks/"+name+".framework/"+name, purego.RTLD_LAZY|purego.RTLD_LOCAL); err != nil {
			return nil, err
		}
	}
	cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return nil, err
	}
	runtimeLibrary, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return nil, err
	}
	g := &labelGraphics{}
	purego.RegisterLibFunc(&g.createImage, runtimeLibrary, "objc_msgSend")
	purego.RegisterLibFunc(&g.setSize, runtimeLibrary, "objc_msgSend")
	purego.RegisterLibFunc(&g.setMargin, runtimeLibrary, "objc_msgSend")
	purego.RegisterLibFunc(&g.newConsumer, cg, "CGDataConsumerCreateWithCFData")
	purego.RegisterLibFunc(&g.releaseConsumer, cg, "CGDataConsumerRelease")
	purego.RegisterLibFunc(&g.newPDF, cg, "CGPDFContextCreate")
	purego.RegisterLibFunc(&g.beginPage, cg, "CGPDFContextBeginPage")
	purego.RegisterLibFunc(&g.drawImage, cg, "CGContextDrawImage")
	purego.RegisterLibFunc(&g.setInterpolation, cg, "CGContextSetInterpolationQuality")
	purego.RegisterLibFunc(&g.endPage, cg, "CGPDFContextEndPage")
	purego.RegisterLibFunc(&g.closePDF, cg, "CGPDFContextClose")
	purego.RegisterLibFunc(&g.releaseContext, cg, "CGContextRelease")
	purego.RegisterLibFunc(&g.releaseImage, cg, "CGImageRelease")
	return g, nil
}

// macSend 调用 Objective-C 对象的方法
func macSend[T any](object objc.ID, method string, args ...any) T {
	return objc.Send[T](object, objc.RegisterName(method), args...)
}

// macClass 查找系统框架中的 Objective-C 类
func macClass(name string) objc.ID {
	return objc.ID(objc.GetClass(name))
}

// macString 构造供 Objective-C 接口读取的 UTF-8 字符串
func macString(value string) objc.ID {
	return macSend[objc.ID](macClass("NSString"), "stringWithUTF8String:", value)
}

// macData 将 Go 字节复制为 NSData
func macData(data []byte) objc.ID {
	result := macSend[objc.ID](macClass("NSData"), "dataWithBytes:length:", unsafe.SliceData(data), len(data))
	runtime.KeepAlive(data)
	return result
}

// copyMacData 将 NSData 复制到 Go 内存
func copyMacData(data objc.ID) []byte {
	count := macSend[int](data, "length")
	start := macSend[*byte](data, "bytes")
	return bytes.Clone(unsafe.Slice(start, count))
}

// macPool 创建作用域内的 Objective-C 自动释放池
func macPool() func() {
	pool := macSend[objc.ID](macClass("NSAutoreleasePool"), "new")
	return func() { macSend[objc.ID](pool, "drain") }
}

// systemFontMenu 使用字体的本地化名称生成字体菜单
func systemFontMenu() fontMenu {
	defer macPool()()
	manager := macSend[objc.ID](macClass("NSFontManager"), "sharedFontManager")
	return makeFontMenu(func(family string) string {
		local := macSend[objc.ID](manager, "localizedNameForFamily:face:", macString(family), objc.ID(0))
		return cmp.Or(macSend[string](local, "UTF8String"), family)
	})
}

// symbolImage 用 Core Image 生成原始条码或二维码模块
func (g *labelGraphics) symbolImage(kind itemKind, value string) (image.Image, error) {
	defer macPool()()
	if g.ci == 0 {
		useSoftware := macSend[objc.ID](macClass("NSNumber"), "numberWithBool:", true)
		options := macSend[objc.ID](macClass("NSDictionary"), "dictionaryWithObject:forKey:", useSoftware, macString("kCIContextUseSoftwareRenderer"))
		ctx := macSend[objc.ID](macClass("CIContext"), "contextWithOptions:", options)
		g.ci = macSend[objc.ID](ctx, "retain")
	}
	filterName := "CIQRCodeGenerator"
	if kind == kindBarcode {
		filterName = "CICode128BarcodeGenerator"
	}
	filter := macSend[objc.ID](macClass("CIFilter"), "filterWithName:", macString(filterName))
	macSend[objc.ID](filter, "setValue:forKey:", macData([]byte(value)), macString("inputMessage"))
	if kind == kindBarcode {
		quiet := macSend[objc.ID](macClass("NSNumber"), "numberWithInt:", int32(10))
		macSend[objc.ID](filter, "setValue:forKey:", quiet, macString("inputQuietSpace"))
	}
	output := macSend[objc.ID](filter, "outputImage")
	if output == 0 {
		return nil, errors.New(tr("error.encode"))
	}
	extent := macSend[cgRect](output, "extent")
	cgImage := g.createImage(g.ci, objc.RegisterName("createCGImage:fromRect:"), output, extent)
	if cgImage == 0 {
		return nil, errors.New(tr("error.renderBarcode"))
	}
	defer g.releaseImage(cgImage)

	rep := macSend[objc.ID](macClass("NSBitmapImageRep"), "alloc")
	rep = macSend[objc.ID](rep, "initWithCGImage:", cgImage)
	defer macSend[objc.ID](rep, "release")
	data := macSend[objc.ID](rep, "representationUsingType:properties:", 4, objc.ID(0))
	return png.Decode(bytes.NewReader(copyMacData(data)))
}

// pageSize 把标签的毫米宽高换算为 PDF 点
func pageSize(l *label) cgSize {
	return cgSize{W: l.Width * 72 / 25.4, H: l.Height * 72 / 25.4}
}

// pdf 生成尺寸与标签一致的单页 PDF
func (g *labelGraphics) pdf(l *label) ([]byte, error) {
	raster, err := l.render()
	if err != nil {
		return nil, err
	}
	var rasterPNG bytes.Buffer
	if err := png.Encode(&rasterPNG, raster); err != nil {
		return nil, err
	}

	defer macPool()()
	bitmap := macSend[objc.ID](macClass("NSBitmapImageRep"), "imageRepWithData:", macData(rasterPNG.Bytes()))
	imageRef := macSend[uintptr](bitmap, "CGImage")
	if imageRef == 0 {
		return nil, errors.New(tr("error.pdfImage"))
	}
	data := macSend[objc.ID](macClass("NSMutableData"), "data")
	consumer := g.newConsumer(data)
	defer g.releaseConsumer(consumer)
	size := pageSize(l)
	box := cgRect{W: size.W, H: size.H}
	context := g.newPDF(consumer, &box, 0)
	if context == 0 {
		return nil, errors.New(tr("error.pdfPage"))
	}
	defer g.releaseContext(context)
	g.beginPage(context, 0)
	g.setInterpolation(context, 1)
	g.drawImage(context, box, imageRef)
	g.endPage(context)
	g.closePDF(context)
	return copyMacData(data), nil
}

// writePDFFile 原子写入导出的 PDF
func writePDFFile(path string, data []byte) error {
	defer macPool()()
	if !macSend[bool](macData(data), "writeToFile:atomically:", macString(path), true) {
		return errors.New(tr("error.pdfWrite"))
	}
	return nil
}

// print 打开 macOS 打印对话框并在用户确认后提交标签
func (g *labelGraphics) print(l *label, title string, _ uintptr) (bool, error) {
	pdf, err := g.pdf(l)
	if err != nil {
		return false, err
	}
	defer macPool()()
	document := macSend[objc.ID](macClass("PDFDocument"), "alloc")
	document = macSend[objc.ID](document, "initWithData:", macData(pdf))
	defer macSend[objc.ID](document, "release")

	info := macSend[objc.ID](macClass("NSPrintInfo"), "sharedPrintInfo")
	info = macSend[objc.ID](info, "copy")
	defer macSend[objc.ID](info, "release")
	macSend[objc.ID](info, "setOrientation:", 0)
	g.setSize(info, objc.RegisterName("setPaperSize:"), pageSize(l))
	for _, edge := range []string{"Top", "Bottom", "Left", "Right"} {
		g.setMargin(info, objc.RegisterName("set"+edge+"Margin:"), 0)
	}
	macSend[objc.ID](info, "setHorizontallyCentered:", false)
	macSend[objc.ID](info, "setVerticallyCentered:", false)

	operation := macSend[objc.ID](document, "printOperationForPrintInfo:scalingMode:autoRotate:", info, 0, false)
	if operation == 0 {
		return false, errors.New(tr("error.printJob"))
	}
	macSend[objc.ID](operation, "setJobTitle:", macString(title))
	macSend[objc.ID](operation, "setShowsPrintPanel:", true)
	return macSend[bool](operation, "runOperation"), nil
}
