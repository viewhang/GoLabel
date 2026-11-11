package printing

import (
	"errors"
	"structs"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"github.com/pengh/GoLabel/internal/platform/cocoa"
)

// cgSize 表示 Apple 图形接口使用的尺寸
type cgSize struct {
	_    structs.HostLayout
	W, H float64
}

// System 使用 macOS 系统打印面板提交 PDF
type System struct {
	pdf       func(platform.Page) ([]byte, error)
	setSize   func(objc.ID, objc.SEL, cgSize)
	setMargin func(objc.ID, objc.SEL, float64)
}

// OpenSystem 加载 macOS 系统打印所需的 PDFKit 和 Objective-C 接口
func OpenSystem(pdf func(platform.Page) ([]byte, error)) (*System, error) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/PDFKit.framework/PDFKit", purego.RTLD_LAZY|purego.RTLD_LOCAL); err != nil {
		return nil, err
	}
	runtimeLibrary, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_LAZY|purego.RTLD_LOCAL)
	if err != nil {
		return nil, err
	}
	s := &System{pdf: pdf}
	purego.RegisterLibFunc(&s.setSize, runtimeLibrary, "objc_msgSend")
	purego.RegisterLibFunc(&s.setMargin, runtimeLibrary, "objc_msgSend")
	return s, nil
}

// Submit 打开 macOS 打印对话框并在用户确认后提交标签
func (s *System) Submit(job Job, _ uintptr) (bool, error) {
	pdf, err := s.pdf(job.Page)
	if err != nil {
		return false, err
	}
	defer cocoa.Pool()()
	document := cocoa.Send[objc.ID](cocoa.Class("PDFDocument"), "alloc")
	document = cocoa.Send[objc.ID](document, "initWithData:", cocoa.Data(pdf))
	defer cocoa.Send[objc.ID](document, "release")

	info := cocoa.Send[objc.ID](cocoa.Class("NSPrintInfo"), "sharedPrintInfo")
	info = cocoa.Send[objc.ID](info, "copy")
	defer cocoa.Send[objc.ID](info, "release")
	cocoa.Send[objc.ID](info, "setOrientation:", 0)
	width, height := job.Page.SizePoints()
	s.setSize(info, objc.RegisterName("setPaperSize:"), cgSize{W: width, H: height})
	for _, edge := range []string{"Top", "Bottom", "Left", "Right"} {
		s.setMargin(info, objc.RegisterName("set"+edge+"Margin:"), 0)
	}
	cocoa.Send[objc.ID](info, "setHorizontallyCentered:", false)
	cocoa.Send[objc.ID](info, "setVerticallyCentered:", false)

	operation := cocoa.Send[objc.ID](document, "printOperationForPrintInfo:scalingMode:autoRotate:", info, 0, false)
	if operation == 0 {
		return false, errors.New(i18n.Tr("error.printJob"))
	}
	cocoa.Send[objc.ID](operation, "setJobTitle:", cocoa.String(job.Title))
	cocoa.Send[objc.ID](operation, "setShowsPrintPanel:", true)
	return cocoa.Send[bool](operation, "runOperation"), nil
}
