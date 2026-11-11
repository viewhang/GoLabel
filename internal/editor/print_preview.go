package editor

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
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
		e.status = i18n.Tr("error.emptyLabel")
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
						BorderWidth(1).BorderColor(colors.Divider).Background(colors.Stage).
						Clip().Draw(e.paintPrintPreview)
				})
				e.printChecks(c.Key("checks"))
			})
			hairline(c.Key("rule-2"))
			ui.Row(c.Key("actions")).Gap(10).AlignItems(ui.Center).Children(func() {
				outlineAction(c.Key("pdf"), i18n.Tr("print.export"), iconDownload, true, func() {
					e.preview.open = false
					e.perform(e.exportPDF)
				})
				ui.Box(c.Key("spacer")).Grow(1)
				plainAction(c.Key("cancel"), i18n.Tr("print.cancel"), func() { e.preview.open = false })
				accentAction(c.Key("print"), i18n.Tr("print.print"), iconPrint, true, func() {
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
			ui.Text(c, i18n.Tr("print.title")).FontSize(19).FontWeight(650).TextColor(colors.Heading)
			ui.Textf(c, "%s · %g × %g mm · %d DPI", e.documentName(), e.label.Width, e.label.Height, e.label.DPI).
				FontSize(11).SingleLine().TextColor(colors.Muted)
		})
		iconAction(c.Key("close"), i18n.Tr("print.close"), iconClose, func() { e.preview.open = false })
	})
}

// printChecks 显示缩放比例和打印前检查结果
func (e *editor) printChecks(c *ui.Context) {
	ui.Column(c).Width(printSettingsWidth).Shrink(0).Gap(12).Children(func() {
		sectionHeading(c, i18n.Tr("print.scale"))
		ui.Row(c.Key("scale")).Height(36).Padding(0, 10).Gap(8).Radius(8).
			BorderWidth(1).BorderColor(colors.Hairline).Background(colors.LayerChip).
			AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPaper).Size(13, 13).Shrink(0).TextColor(colors.Muted)
			ui.Text(c, i18n.Tr("print.actualSize")).FontSize(12).FontWeight(600).SingleLine().
				Grow(1).TextColor(colors.Text)
		})

		hairline(c.Key("rule"))

		sectionHeading(c, i18n.Tr("print.checks"))
		ui.Column(c.Key("items")).Gap(7).Children(func() {
			for i, check := range e.preflights() {
				ui.Row(c.Key(i)).Gap(8).AlignItems(ui.Start).Children(func() {
					ink, fill := colors.Success, colors.SuccessSoft
					if !check.ok {
						ink, fill = colors.Warning, colors.WarningSoft
					}
					ui.Row(c.Key("mark")).Size(16, 16).Radius(9).Shrink(0).
						Background(fill).Justify(ui.Center).AlignItems(ui.Center).
						Children(func() {
							icon := iconCheck
							if !check.ok {
								icon = iconClose
							}
							ui.Icon(c, icon).Size(10, 10).TextColor(ink)
						})
					ui.Text(c, check.text).FontSize(11).Grow(1).TextColor(colors.Body)
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
	modules, modulesOK := i18n.Tr("print.noCodes"), true
	bounded, boundedOK := i18n.Tr("print.inside"), true
	for i := range l.Items {
		a := &l.Items[i]
		if a.Kind != kindBarcode && a.Kind != kindQR {
			continue
		}
		modules = i18n.Tr("print.modules")
		if e.graphics == nil {
			// 没有图形上下文时无法测量码的点阵，只做边界检查
			continue
		}
		// 逐项检查内容：缺码值或模块过窄都会让打印失败
		if a.symbol == nil && a.issue == nil {
			a.prepareSymbol(e.graphics)
		}
		if a.symbol == nil || a.moduleWidth(a.box(l, page)) < 2 {
			modules, modulesOK = i18n.Trf("print.narrow", i+1, kindTitle(a.Kind)), false
		}
	}
	for i := range l.Items {
		a := &l.Items[i]
		if a.X < 0 || a.Y < 0 || a.X+a.W > l.Width+1e-8 || a.Y+a.H > l.Height+1e-8 {
			bounded, boundedOK = i18n.Trf("print.outside", i+1, kindTitle(a.Kind)), false
		}
	}
	return []preflight{
		{text: modules, ok: modulesOK},
		{text: bounded, ok: boundedOK},
		{text: i18n.Trf("print.resolution", l.DPI), ok: true},
	}
}

// paintPrintPreview 在预览区按比例绘制标签内容
func (e *editor) paintPrintPreview(p *ui.Painter, r ui.Rect) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	p.Fill(r, colors.Stage, 0)
	for y := r.Y + 8; y < r.Y+r.H; y += 16 {
		for x := r.X + 8; x < r.X+r.W; x += 16 {
			p.Fill(ui.Rect{X: x - 1, Y: y - 1, W: 2, H: 2}, colors.SketchDot, 1)
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
	p.Shadow(paper, 0, 0, 12, 32, -10, colors.ShadowDeep)
	l.paint(p, paper, false)
	p.Stroke(paper, ui.Hex("#D5DEE8"), 0, 1)
}
