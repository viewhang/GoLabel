package platform

import (
	"bytes"
	"cmp"
	"errors"
	"image"
	"image/png"
	"structs"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform/cocoa"
)

// cgRect 表示 Apple 图形接口使用的矩形
type cgRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// Graphics 持有条码生成器和 PDF 绘制接口
type Graphics struct {
	ci objc.ID

	createImage func(objc.ID, objc.SEL, objc.ID, cgRect) uintptr

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

// OpenGraphics 加载条码生成和 PDF 绘制所需的 Apple 框架
func OpenGraphics() (*Graphics, error) {
	for _, name := range []string{"AppKit", "CoreImage"} {
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
	g := &Graphics{}
	purego.RegisterLibFunc(&g.createImage, runtimeLibrary, "objc_msgSend")
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

// copyMacData 将 NSData 复制到 Go 内存
func copyMacData(data objc.ID) []byte {
	count := cocoa.Send[int](data, "length")
	start := cocoa.Send[*byte](data, "bytes")
	return bytes.Clone(unsafe.Slice(start, count))
}

// FontFamilyName 返回 macOS 的本地化字体族名
func FontFamilyName(family string) string {
	defer cocoa.Pool()()
	manager := cocoa.Send[objc.ID](cocoa.Class("NSFontManager"), "sharedFontManager")
	local := cocoa.Send[objc.ID](manager, "localizedNameForFamily:face:", cocoa.String(family), objc.ID(0))
	return cmp.Or(cocoa.Send[string](local, "UTF8String"), family)
}

// SymbolImage 用 Core Image 生成原始条码或二维码模块
func (g *Graphics) SymbolImage(code128Barcode bool, value string) (image.Image, error) {
	defer cocoa.Pool()()
	if g.ci == 0 {
		useSoftware := cocoa.Send[objc.ID](cocoa.Class("NSNumber"), "numberWithBool:", true)
		options := cocoa.Send[objc.ID](cocoa.Class("NSDictionary"), "dictionaryWithObject:forKey:", useSoftware, cocoa.String("kCIContextUseSoftwareRenderer"))
		ctx := cocoa.Send[objc.ID](cocoa.Class("CIContext"), "contextWithOptions:", options)
		g.ci = cocoa.Send[objc.ID](ctx, "retain")
	}
	filterName := "CIQRCodeGenerator"
	if code128Barcode {
		filterName = "CICode128BarcodeGenerator"
	}
	filter := cocoa.Send[objc.ID](cocoa.Class("CIFilter"), "filterWithName:", cocoa.String(filterName))
	cocoa.Send[objc.ID](filter, "setValue:forKey:", cocoa.Data([]byte(value)), cocoa.String("inputMessage"))
	if code128Barcode {
		quiet := cocoa.Send[objc.ID](cocoa.Class("NSNumber"), "numberWithInt:", int32(10))
		cocoa.Send[objc.ID](filter, "setValue:forKey:", quiet, cocoa.String("inputQuietSpace"))
	}
	output := cocoa.Send[objc.ID](filter, "outputImage")
	if output == 0 {
		return nil, errors.New(i18n.Tr("error.encode"))
	}
	extent := cocoa.Send[cgRect](output, "extent")
	cgImage := g.createImage(g.ci, objc.RegisterName("createCGImage:fromRect:"), output, extent)
	if cgImage == 0 {
		return nil, errors.New(i18n.Tr("error.renderBarcode"))
	}
	defer g.releaseImage(cgImage)

	rep := cocoa.Send[objc.ID](cocoa.Class("NSBitmapImageRep"), "alloc")
	rep = cocoa.Send[objc.ID](rep, "initWithCGImage:", cgImage)
	defer cocoa.Send[objc.ID](rep, "release")
	data := cocoa.Send[objc.ID](rep, "representationUsingType:properties:", 4, objc.ID(0))
	return png.Decode(bytes.NewReader(copyMacData(data)))
}

// PDF 生成尺寸与标签一致的单页 PDF
func (g *Graphics) PDF(page Page) ([]byte, error) {
	var rasterPNG bytes.Buffer
	if err := png.Encode(&rasterPNG, page.Raster); err != nil {
		return nil, err
	}

	defer cocoa.Pool()()
	bitmap := cocoa.Send[objc.ID](cocoa.Class("NSBitmapImageRep"), "imageRepWithData:", cocoa.Data(rasterPNG.Bytes()))
	imageRef := cocoa.Send[uintptr](bitmap, "CGImage")
	if imageRef == 0 {
		return nil, errors.New(i18n.Tr("error.pdfImage"))
	}
	data := cocoa.Send[objc.ID](cocoa.Class("NSMutableData"), "data")
	consumer := g.newConsumer(data)
	defer g.releaseConsumer(consumer)
	width, height := page.SizePoints()
	box := cgRect{W: width, H: height}
	context := g.newPDF(consumer, &box, 0)
	if context == 0 {
		return nil, errors.New(i18n.Tr("error.pdfPage"))
	}
	defer g.releaseContext(context)
	g.beginPage(context, 0)
	g.setInterpolation(context, 1)
	g.drawImage(context, box, imageRef)
	g.endPage(context)
	g.closePDF(context)
	return copyMacData(data), nil
}

// WritePDFFile 原子写入导出的 PDF
func WritePDFFile(path string, data []byte) error {
	defer cocoa.Pool()()
	if !cocoa.Send[bool](cocoa.Data(data), "writeToFile:atomically:", cocoa.String(path), true) {
		return errors.New(i18n.Tr("error.pdfWrite"))
	}
	return nil
}
