package main

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

const (
	// printDialogWidth 是打印预览对话框的内容宽度
	printDialogWidth = 900
	// printPreviewHeight 是打印预览区的高度
	printPreviewHeight = 392
	// printSettingsWidth 是对话框右侧检查栏的宽度
	printSettingsWidth = 300
)

// printPreview 保存打印预览对话框的打开状态
type printPreview struct {
	open bool
}

// preflight 是打印前检查中的一项
type preflight struct {
	text   string
	ok     bool
	detail string
}

// openPrintPreview 在标签非空时打开打印预览
func (e *editor) openPrintPreview() {
	if len(e.label.Items) == 0 {
		e.status = "请先添加标签内容"
		return
	}
	e.preview.open = true
}

// printPreviewDialog 显示打印预览与打印前检查
func (e *editor) printPreviewDialog(c *ui.Context) {
	if !e.preview.open {
		return
	}
	ui.Modal(c, &e.preview.open, func() {
		ui.Column(c.Key("dialog")).Width(printDialogWidth).Children(func() {
			e.previewHeading(c.Key("heading"))
			hairline(c.Key("rule"))
			ui.Row(c.Key("body")).Gap(24).Padding(16, 0).AlignItems(ui.Start).Children(func() {
				ui.Column(c.Key("sheet")).Grow(1).MinWidth(0).Children(func() {
					ui.Box(c.Key("preview")).Height(printPreviewHeight).Radius(12).
						BorderWidth(1).BorderColor(ui.Hex(colorDivider)).Background(ui.Hex("#F4F6F9")).
						Clip().Draw(e.paintPrintPreview)
				})
				e.printChecks(c.Key("checks"))
			})
			hairline(c.Key("rule-2"))
			ui.Row(c.Key("actions")).Gap(10).AlignItems(ui.Center).Children(func() {
				outlineAction(c.Key("pdf"), "导出 PDF", iconDownload, true, func() {
					e.preview.open = false
					e.perform(e.exportPDF)
				})
				ui.Box(c.Key("spacer")).Grow(1)
				plainAction(c.Key("cancel"), "取消", func() { e.preview.open = false })
				accentAction(c.Key("print"), "打印", iconPrint, true, func() {
					e.preview.open = false
					e.perform(e.print)
				})
			})
		})
	})
}

// previewHeading 显示对话框标题和关闭按钮
func (e *editor) previewHeading(c *ui.Context) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Column(c.Key("titles")).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, "打印预览").FontSize(19).FontWeight(650).TextColor(ui.Hex(colorHeading))
			ui.Textf(c, "%s · %g × %g mm · %d DPI", e.documentName(), e.label.Width, e.label.Height, e.label.DPI).
				FontSize(11).SingleLine().TextColor(ui.Hex(colorMuted))
		})
		iconAction(c.Key("close"), "关闭", iconClose, func() { e.preview.open = false })
	})
}

// printChecks 显示缩放比例和打印前检查结果
func (e *editor) printChecks(c *ui.Context) {
	ui.Column(c).Width(printSettingsWidth).Shrink(0).Gap(12).Children(func() {
		sectionHeading(c, "缩放")
		ui.Row(c.Key("scale")).Height(36).Padding(0, 10).Gap(8).Radius(8).
			BorderWidth(1).BorderColor(ui.Hex(colorHairline)).Background(ui.Hex(colorLayerChip)).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPaper).Size(13, 13).Shrink(0).TextColor(ui.Hex(colorMuted))
			ui.Text(c, "100%（按实际尺寸打印）").FontSize(12).FontWeight(600).SingleLine().
				Grow(1).TextColor(ui.Hex(colorText))
		})

		hairline(c.Key("rule"))

		sectionHeading(c, "检查")
		ui.Column(c.Key("items")).Gap(7).Children(func() {
			for i, check := range e.preflights() {
				ui.Row(c.Key(i)).Gap(8).AlignItems(ui.Start).Children(func() {
					ink, fill := colorSuccess, colorSuccessSoft
					if !check.ok {
						ink, fill = colorWarning, colorWarningSoft
					}
					ui.Row(c.Key("mark")).Size(16, 16).Radius(9).Shrink(0).
						Background(ui.Hex(fill)).Justify(ui.Center).AlignItems(ui.Center).
						Children(func() {
							icon := iconCheck
							if !check.ok {
								icon = iconClose
							}
							ui.Icon(c, icon).Size(10, 10).TextColor(ui.Hex(ink))
						})
					ui.Text(c, check.text).FontSize(11).Grow(1).TextColor(ui.Hex(colorBody))
				})
			}
		})
	})
}

// preflights 计算打印前的检查项：码的点阵、元素边界和分辨率
func (e *editor) preflights() []preflight {
	l := e.label
	width := int(math.Round(l.Width * float64(l.DPI) / 25.4))
	height := int(math.Round(l.Height * float64(l.DPI) / 25.4))
	page := ui.Rect{W: float32(width), H: float32(height)}
	modules, modulesOK := "未使用条码或二维码", true
	bounded, boundedOK := "元素都在标签内", true
	for i := range l.Items {
		a := &l.Items[i]
		if a.Kind != kindBarcode && a.Kind != kindQR {
			continue
		}
		modules = "条码模块 ≥ 2 点"
		if e.graphics == nil {
			// 没有图形上下文时无法测量码的点阵，只做边界检查
			continue
		}
		// 逐项检查内容：缺码值或模块过窄都会让打印失败
		if a.symbol == nil && a.issue == nil {
			a.prepareSymbol(e.graphics)
		}
		if a.symbol == nil || a.moduleWidth(a.box(l, page)) < 2 {
			modules, modulesOK = fmt.Sprintf("第 %d 个%s模块过窄", i+1, kindTitle(a.Kind)), false
		}
	}
	for i := range l.Items {
		a := &l.Items[i]
		if a.X < 0 || a.Y < 0 || a.X+a.W > l.Width+1e-8 || a.Y+a.H > l.Height+1e-8 {
			bounded, boundedOK = fmt.Sprintf("第 %d 个%s超出标签边界", i+1, kindTitle(a.Kind)), false
		}
	}
	return []preflight{
		{text: modules, ok: modulesOK},
		{text: bounded, ok: boundedOK},
		{text: fmt.Sprintf("分辨率 %d DPI", l.DPI), ok: true},
	}
}

// paintPrintPreview 在预览区按比例绘制标签内容
func (e *editor) paintPrintPreview(p *ui.Painter, r ui.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	p.Fill(r, ui.Hex("#F4F6F9"), 0)
	for y := r.Y + 8; y < r.Y+r.H; y += 16 {
		for x := r.X + 8; x < r.X+r.W; x += 16 {
			p.Fill(ui.Rect{X: x - 1, Y: y - 1, W: 2, H: 2}, ui.Hex("#DCE3EB"), 1)
		}
	}
	l := e.label
	scale := min((r.W-48)/float32(l.Width), (r.H-48)/float32(l.Height))
	paper := ui.Rect{
		X: r.X + (r.W-float32(l.Width)*scale)/2,
		Y: r.Y + (r.H-float32(l.Height)*scale)/2,
		W: float32(l.Width) * scale,
		H: float32(l.Height) * scale,
	}
	p.Shadow(paper, 0, 0, 12, 32, -10, ui.RGBA(16, 24, 40, 0.26))
	l.paint(p, paper, false)
	p.Stroke(paper, ui.Hex("#D5DEE8"), 0, 1)
}
