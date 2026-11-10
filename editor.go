package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

const (
	// minCanvasZoom 是画布允许的最小缩放比例
	minCanvasZoom = 0.5
	// maxCanvasZoom 是画布允许的最大缩放比例
	maxCanvasZoom = 2
	// alignmentDistance 是触发对齐吸附的屏幕距离
	alignmentDistance = 5
	// canvasDotStep 是画布点阵的屏幕间距
	canvasDotStep = 22
	// canvasDotSize 是画布点阵圆点的屏幕边长
	canvasDotSize = 2
	// panelPadding 是两侧面板的左右内边距
	panelPadding = 16
	// sidebarWidth 是左侧元素与图层面板的宽度
	sidebarWidth = 260
	// inspectorWidth 是右侧属性栏的宽度
	inspectorWidth = 320
	// handleSize 是选中框四角手柄的屏幕边长
	handleSize = 8
)

// editor 保存一个标签窗口的选择、平移、缩放和交互状态
type editor struct {
	document  *mygo.Document[*label]
	documents *mygo.Documents[*label]
	graphics  *labelGraphics
	window    *mygo.Window
	label     *label
	selected  int
	zoom      float32
	pan       ui.Point
	canvas    ui.Handle
	drag      *gesture
	textEdit  *textEditState
	inline    bool
	cells     *tableSelection
	status    string
	title     string
	form      newLabelForm
	preview   printPreview
	settings  settingsState
}

// gesture 记录元素拖动或单元格拖选的起点状态
type gesture struct {
	item           item
	startX, startY float32
	resize         bool
	corner         int
	guideX, guideY alignmentGuide
	selectCells    bool
	previousCells  *tableSelection
}

// alignmentGuide 保存标签毫米坐标中的一条对齐线
type alignmentGuide struct {
	at      float64
	visible bool
}

// content 返回当前标签；未绑定文档时使用内存中的标签
func (e *editor) content() *label {
	if e.document != nil {
		return e.document.Content()
	}
	return e.label
}

// documentName 返回窗口顶部显示的标签名
func (e *editor) documentName() string {
	if e.document != nil && e.document.Path() != "" {
		return e.document.Name()
	}
	if e.title != "" {
		return e.title
	}
	if e.document == nil {
		return "未命名标签"
	}
	return e.document.Name()
}

// documentEdited 报告标签是否有未保存的修改
func (e *editor) documentEdited() bool {
	return e.document != nil && e.document.IsEdited()
}

// changed 标记标签内容已修改；未绑定文档时只保留内存中的修改
func (e *editor) changed() {
	if e.document != nil {
		e.document.Changed()
	}
}

// view 构建顶部工具栏、左侧元素与图层、画布和右侧属性栏
func (e *editor) view(c *ui.Context) {
	c.SetTheme(appTheme())
	l := e.content()
	if l != e.label {
		e.label, e.selected, e.drag = l, -1, nil
		e.textEdit, e.cells, e.inline = nil, nil, false
		e.title = ""
		e.form.open, e.preview.open, e.settings.open = false, false, false
		for i := range l.Items {
			element := &l.Items[i]
			if e.graphics != nil && (element.Kind == kindQR || element.Kind == kindBarcode) {
				element.prepareSymbol(e.graphics)
			}
		}
	}

	c.OnShortcut(ui.Cmd, ui.KeyP, func() { e.openPrintPreview() })
	c.OnShortcut(ui.Cmd, ui.KeyComma, func() { e.openSettings(settingsGeneral) })
	ui.Column(c).Fill().Background(ui.Hex(colorCanvas)).Children(func() {
		e.header(c.Key("header"))
		ui.Row(c.Key("workspace")).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			e.sidebar(c.Key("sidebar"))
			e.canvasView(c.Key("canvas"))
			e.inspector(c.Key("inspector"))
		})
	})
	e.newLabelDialog(c)
	e.printPreviewDialog(c)
	e.settingsDialog(c)
}

// header 显示品牌、文件操作、标签名以及导出和打印按钮
func (e *editor) header(c *ui.Context) {
	l := e.label
	ui.Row(c).Height(62).Shrink(0).Padding(0, 20).Gap(16).AlignItems(ui.Center).
		Background(ui.Hex(colorPanel)).BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex(colorDivider)).
		Children(func() {
			ui.Row(c.Key("brand")).Gap(10).Shrink(0).AlignItems(ui.Center).Children(func() {
				ui.Row(c.Key("mark")).Size(30, 30).Radius(8).Shrink(0).Background(ui.Hex(colorAccent)).
					Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconLogo).Size(17, 17).TextColor(ui.Hex(colorPanel))
				})
				ui.Text(c, "GoLabel").FontSize(14.5).FontWeight(650).TextColor(ui.Hex(colorHeading))
			})
			ui.Box(c.Key("rule")).Width(1).Height(20).Shrink(0).Background(ui.Hex(colorDivider))

			ui.Row(c.Key("file")).Gap(1).Shrink(0).Children(func() {
				ghostAction(c.Key("new"), "新建", iconPlus, func() { e.form.openNewLabel() })
				ghostAction(c.Key("open"), "打开", iconFolder, func() {
					e.perform(func() error {
						_, err := e.documents.OpenDialog()
						return err
					})
				})
				ghostAction(c.Key("save"), "保存", iconSave, func() {
					e.perform(e.document.Save)
				})
			})

			ui.Box(c.Key("spacer")).Grow(1)
			empty := len(l.Items) == 0
			ui.Row(c.Key("actions")).Gap(8).Shrink(0).AlignItems(ui.Center).Children(func() {
				ghostAction(c.Key("templates"), "模板", iconGrid, func() { e.openSettings(settingsTemplates) })
				ghostAction(c.Key("settings"), "设置", iconSliders, func() { e.openSettings(settingsGeneral) })
				outlineAction(c.Key("pdf"), "导出 PDF", iconDownload, !empty, func() {
					e.perform(e.exportPDF)
				})
				accentAction(c.Key("print"), "打印", iconPrint, !empty, func() {
					e.openPrintPreview()
				})
			})

			e.documentTitle(c.Key("title"))
		})
}

// documentTitle 在顶栏中央显示标签名和保存状态
func (e *editor) documentTitle(c *ui.Context) {
	ui.Row(c).Absolute().LeftPercent(50).Top(0).Height(62).TranslatePercent(-50, 0).
		Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, e.documentName()).FontSize(13).FontWeight(600).MaxWidth(240).
			SingleLine().TextColor(ui.Hex(colorHeading))
		text, ink, fill := "已保存", colorSuccess, colorSuccessSoft
		switch {
		case e.status != "":
			text, ink, fill = e.status, colorMuted, colorHover
		case e.documentEdited():
			text, ink, fill = "未保存", colorWarning, colorWarningSoft
		}
		ui.Row(c.Key("state")).Height(20).Padding(0, 7).Gap(5).Radius(10).Shrink(0).
			Background(ui.Hex(fill)).AlignItems(ui.Center).Children(func() {
			ui.Box(c.Key("dot")).Size(5, 5).Radius(3).Shrink(0).Background(ui.Hex(ink))
			ui.Text(c, text).FontSize(10.5).FontWeight(500).SingleLine().MaxWidth(180).TextColor(ui.Hex(ink))
		})
	})
}

// ghostAction 显示顶栏中无边框的图标按钮
func ghostAction(c *ui.Context, label string, icon *ui.SVG, action func()) {
	button := ui.ButtonBase(c).Height(32).Padding(0, 9).Radius(7).Gap(6).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.Background(ui.Hex(colorHover))
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(ui.Hex(colorBody))
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(ui.Hex(colorBody))
	})
	if button.Clicked() {
		action()
	}
}

// outlineAction 显示顶栏中描边的次要按钮
func outlineAction(c *ui.Context, label string, icon *ui.SVG, enabled bool, action func()) {
	ink, border, fill := colorBody, colorFieldBorder, colorPanel
	if !enabled {
		ink, border, fill = "#B3BCCB", colorDivider, colorFieldBg
	}
	button := ui.ButtonBase(c).Height(34).Padding(0, 12).Radius(8).Gap(6).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Disabled(!enabled).
		BorderWidth(1).BorderColor(ui.Hex(border)).Background(ui.Hex(fill))
	if button.Hovered() && enabled {
		button.Background(ui.Hex(colorFieldBg)).BorderColor(ui.Hex("#C9D3DF"))
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(ui.Hex(ink))
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(ui.Hex(ink))
	})
	if button.Clicked() && enabled {
		action()
	}
}

// accentAction 显示顶栏中的主色按钮
func accentAction(c *ui.Context, label string, icon *ui.SVG, enabled bool, action func()) {
	fill, ink := colorAccent, colorPanel
	if !enabled {
		fill, ink = "#E4E9F0", "#9AA6B6"
	}
	button := ui.ButtonBase(c).Height(34).Padding(0, 14).Radius(8).Gap(7).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Disabled(!enabled).
		Background(ui.Hex(fill)).Shadow(0, 1, 3, 0, ui.RGBA(16, 24, 40, 0.16))
	if button.Hovered() && enabled {
		button.Background(ui.Hex("#2743C4"))
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(ui.Hex(ink))
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(ui.Hex(ink))
	})
	if button.Clicked() && enabled {
		action()
	}
}

// sidebar 显示添加元素、图层列表和纸张设置
func (e *editor) sidebar(c *ui.Context) {
	l := e.label
	ui.Column(c).Width(sidebarWidth).Shrink(0).FillHeight().Background(ui.Hex(colorPanel)).
		BorderWidth(0, 1, 0, 0).BorderColor(ui.Hex(colorDivider)).Children(func() {
		ui.Row(c.Key("add-head")).Padding(18, panelPadding, 10, panelPadding).Shrink(0).Children(func() {
			sectionHeading(c, "添加元素")
		})
		ui.Column(c.Key("tools")).Padding(0, panelPadding, 16, panelPadding).Shrink(0).Gap(8).Children(func() {
			tools := [...]struct {
				kind itemKind
				name string
				icon *ui.SVG
			}{
				{kindText, "文字", iconText}, {kindImage, "图片", iconPicture},
				{kindBarcode, "条码", iconBarcode}, {kindQR, "二维码", iconQR},
				{kindLine, "线条", iconLine}, {kindTable, "表格", iconTable},
			}
			for i := 0; i < len(tools); i += 2 {
				ui.Row(c.Key(i)).Gap(8).Children(func() {
					for _, tool := range tools[i:min(i+2, len(tools))] {
						e.toolCard(c.Key(tool.kind), tool.kind, tool.name, tool.icon)
					}
				})
			}
		})

		hairline(c.Key("divider"))

		ui.Row(c.Key("layer-head")).Padding(14, panelPadding, 8, panelPadding).Gap(7).Shrink(0).AlignItems(ui.Center).Children(func() {
			sectionHeading(c, "图层")
			ui.Row(c.Key("count")).Height(17).MinWidth(17).Padding(0, 5).Radius(9).Shrink(0).
				Background(ui.Hex(colorHover)).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d", len(l.Items)).FontSize(10).FontWeight(600).TextColor(ui.Hex(colorMuted))
			})
		})

		ui.Scroll(c.Key("layers")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key("rows")).Padding(0, 8, 8, 8).Gap(2).Children(func() {
				if len(l.Items) == 0 {
					emptyLayers(c.Key("empty"))
					return
				}
				for i := len(l.Items) - 1; i >= 0; i-- {
					e.layerRow(c.Key(i), i)
				}
			})
		})

		ui.Column(c.Key("paper-settings")).Padding(8, 14, 14, 14).Shrink(0).Children(func() {
			e.paperSettings(c.Key("paper"))
		})
	})
}

// toolCard 显示可点击或拖入画布的元素工具卡片
func (e *editor) toolCard(c *ui.Context, kind itemKind, name string, icon *ui.SVG) {
	ink := ui.Hex(colorBody)
	card := ui.ButtonBase(c).Grow(1).Height(60).Radius(10).Column().Gap(7).
		AlignItems(ui.Center).Justify(ui.Center).Role(ui.RoleButton).Label(name).
		Cursor(ui.CursorPointer).Drag(kind).
		BorderWidth(1).BorderColor(ui.Hex("#E8ECF2")).Background(ui.Hex(colorFieldBg))
	if card.Hovered() {
		card.BorderColor(ui.Hex("#C7D2E3")).Background(ui.Hex(colorPanel))
		ink = ui.Hex(colorAccent)
	}
	card.Children(func() {
		ui.Icon(c, icon).Size(19, 19).TextColor(ink)
		ui.Text(c, name).FontSize(12).FontWeight(500).TextColor(ui.Hex(colorBody))
	})
	if x, y, ok := e.toolPosition(c, card); ok {
		if kind == kindImage {
			e.perform(func() error { return e.importImage(x, y) })
		} else {
			e.add(item{Kind: kind, X: x, Y: y})
		}
	}
}

// toolPosition 将添加工具的点击或拖放落点换算为标签毫米坐标
func (e *editor) toolPosition(c *ui.Context, button ui.Element) (x, y float64, ok bool) {
	if button.Clicked() {
		return 2, 2, true
	}
	point, dropped := button.DragMissed()
	if !dropped {
		return 0, 0, false
	}
	bounds := e.canvas.Bounds(c)
	paper := e.paperRect(bounds)
	if !bounds.Contains(point.X, point.Y) || !paper.Contains(point.X, point.Y) {
		return 0, 0, false
	}
	x = roundMM(float64(point.X-paper.X) / float64(paper.W) * e.label.Width)
	y = roundMM(float64(point.Y-paper.Y) / float64(paper.H) * e.label.Height)
	return x, y, true
}

// layerRow 显示一个图层的图标、名称和毫米尺寸
func (e *editor) layerRow(c *ui.Context, index int) {
	a := &e.label.Items[index]
	selected := index == e.selected
	ink, name, size := colorFaint, colorBody, colorFaint
	chip, chipBorder := colorLayerChip, colorHairline
	if selected {
		ink, name, size = colorAccent, colorText, colorMuted
		chip, chipBorder = colorPanel, colorFieldBorder
	}
	weight := 500
	if selected {
		weight = 600
	}
	row := ui.ButtonBase(c).Height(38).FillWidth().Padding(0, 8).Radius(7).Gap(9).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(layerName(a)).AccessSelected(selected)
	switch {
	case selected:
		row.Background(ui.Hex(colorSelected))
	case row.Hovered():
		row.Background(ui.Hex(colorLayerHover))
	}
	row.Children(func() {
		ui.Row(c.Key("icon")).Size(22, 22).Radius(5).Shrink(0).BorderWidth(1).
			BorderColor(ui.Hex(chipBorder)).Background(ui.Hex(chip)).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, layerIcon(a.Kind)).Size(13, 13).TextColor(ui.Hex(ink))
		})
		ui.Text(c, layerName(a)).FontSize(12.5).FontWeight(weight).SingleLine().
			TextColor(ui.Hex(name)).Grow(1)
		ui.Text(c, fmt.Sprintf("%g × %g", a.W, a.H)).FontSize(10).Shrink(0).TextColor(ui.Hex(size))
	})
	if row.Clicked() {
		e.selectItem(index)
	}
}

// layerName 返回图层列表中显示的元素名称
func layerName(a *item) string {
	if a.Kind == kindText {
		if name := []rune(a.textContent().PlainText()); len(name) > 0 {
			return string(name[:min(12, len(name))])
		}
		return "文字"
	}
	return kindTitle(a.Kind)
}

// emptyLayers 在没有元素时提示从左侧添加
func emptyLayers(c *ui.Context) {
	ui.Column(c).Margin(0, 6, 0, 0).Padding(26, 14).Radius(10).Gap(10).
		BorderWidth(1).BorderStyle(ui.BorderDashed).BorderColor(ui.Hex("#DDE5EF")).
		Background(ui.Hex(colorFieldBg)).AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("badge")).Size(34, 34).Radius(8).Background(ui.Hex("#F2F5F9")).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconStack).Size(17, 17).TextColor(ui.Hex(colorRulerText))
		})
		ui.Text(c, "暂无元素").FontSize(11.5).FontWeight(600).TextColor(ui.Hex("#475467"))
	})
}

// paperSettings 以规格卡片打开纸张尺寸和 DPI 设置
func (e *editor) paperSettings(c *ui.Context) {
	l := e.label
	button := ui.ButtonBase(c).Height(52).FillWidth().Padding(0, 11).Gap(10).Radius(10).
		AlignItems(ui.Center).Role(ui.RoleButton).Label("标签纸张设置").Cursor(ui.CursorPointer).
		BorderWidth(1).BorderColor(ui.Hex(colorDivider)).Background(ui.Hex(colorLayerChip))
	if button.Hovered() {
		button.Background(ui.Hex("#F2F5F9")).BorderColor(ui.Hex("#D6DEE8"))
	}
	button.Children(func() {
		ui.Row(c.Key("icon")).Size(30, 30).Radius(7).Shrink(0).BorderWidth(1).
			BorderColor(ui.Hex(colorFieldBorder)).Background(ui.Hex(colorPanel)).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPaper).Size(16, 16).TextColor(ui.Hex("#475467"))
		})
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Textf(c, "%g × %g mm", l.Width, l.Height).FontSize(12).FontWeight(600).TextColor(ui.Hex(colorText))
			ui.Textf(c, "%d DPI", l.DPI).FontSize(10).TextColor(ui.Hex(colorMuted))
		})
		ui.Icon(c, iconChevron).Size(14, 14).Shrink(0).TextColor(ui.Hex(colorFaint))
	})

	open := ui.Local(button, "open", func() bool { return false })
	if button.Clicked() {
		*open = !*open
	}
	ui.Popover(c.Key("popover"), button, open, func() {
		ui.Column(c.Key("fields")).Width(228).Padding(12).Gap(10).Children(func() {
			w, h, dpi := l.Width, l.Height, l.DPI
			ui.Row(c.Key("paper-size")).Gap(8).Children(func() {
				inspectorDimension(c, "纸宽", &l.Width, minimumSide, maximumSide)
				inspectorDimension(c, "纸高", &l.Height, minimumSide, maximumSide)
			})
			if w != l.Width || h != l.Height {
				for i := range l.Items {
					l.Items[i].constrain(l)
				}
			}

			resolutions := [...]int{203, 300, 600}
			selectedDPI := slices.Index(resolutions[:], l.DPI)
			ui.Column(c.Key("dpi")).Gap(5).Children(func() {
				ui.Text(c, "打印分辨率").FontSize(10).TextColor(ui.Hex(colorMuted))
				optionRow(c.Key("segments"), []string{"203", "300", "600"}, selectedDPI, func(i int) {
					l.DPI = resolutions[i]
				})
			})
			if w != l.Width || h != l.Height || dpi != l.DPI {
				e.changed()
			}
		})
	})
}

// fitView 恢复适合窗口的缩放与居中
func (e *editor) fitView() {
	e.zoom, e.pan = 1, ui.Point{}
}

// zoomBy 以画布中心为基准调整缩放比例
func (e *editor) zoomBy(scale float32) {
	e.zoom = min(maxCanvasZoom, max(minCanvasZoom, e.zoom*scale))
}

// canvasView 构建画布、辅助标注、浮动工具栏和缩放控件
func (e *editor) canvasView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		canvas := ui.Box(c.Key("surface")).Fill().Clip().Focusable().Label("标签画布").
			Bind(&e.canvas).HandleInput(func(ev ui.InputEvent) bool { return e.handleCanvas(c, ev) }).
			Draw(func(p *ui.Painter, viewport ui.Rect) { e.paintCanvas(p, viewport, t) })
		if e.inline && (canvas.Shortcut(0, ui.KeyEscape) ||
			(e.label.Items[e.selected].Kind == kindTable && canvas.Shortcut(0, ui.KeyEnter))) {
			e.inline = false
			canvas.Focus()
		}
		if e.inline && canvas.FocusLeft() {
			e.inline = false
		}
		canvas.Children(func() {
			if e.inline {
				e.inlineEditor(c.Key("inline-text"), canvas)
			}
		})

		bounds := canvas.Bounds()
		viewport := ui.Rect{W: bounds.W, H: bounds.H}
		e.trackViewport(c, canvas, viewport)
		paper := e.paperRect(viewport)
		if len(e.label.Items) == 0 && !e.inline {
			e.emptyPaper(c.Key("empty-paper"), paper)
		}
		if a := e.selectedItem(); a != nil && a.Kind == kindText && !e.inline {
			e.textToolbar(c.Key("text-toolbar"), a, paper)
		}
		e.heightRuler(c.Key("height-ruler"), paper, viewport)
		e.zoomControl(c.Key("zoom"), viewport)
	})
}

// trackViewport 在画布尺寸变化后请求重绘，让浮层跟上新的布局
func (e *editor) trackViewport(c *ui.Context, canvas ui.Element, viewport ui.Rect) {
	last := ui.Local(canvas, "viewport", func() ui.Point { return ui.Point{} })
	size := ui.Point{X: viewport.W, Y: viewport.H}
	if *last != size {
		*last = size
		c.After(0)
	}
}

// selectedItem 返回当前选中的元素，没有选中时返回 nil
func (e *editor) selectedItem() *item {
	if e.selected < 0 || e.selected >= len(e.label.Items) {
		return nil
	}
	return &e.label.Items[e.selected]
}

// emptyPaper 在空白标签上提示添加文字或图片
func (e *editor) emptyPaper(c *ui.Context, paper ui.Rect) {
	ui.Column(c).Absolute().Left(paper.X).Top(paper.Y).Size(paper.W, paper.H).Padding(26).
		PassThrough().Children(func() {
		ui.Column(c.Key("card")).Fill().Gap(18).Radius(6).PassThrough().
			BorderWidth(1.5).BorderStyle(ui.BorderDashed).BorderColor(ui.Hex("#D7E0EA")).
			Background(ui.Hex("#FCFDFE")).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Row(c.Key("badge")).Size(48, 48).Radius(12).Background(ui.Hex("#F2F5F9")).
				Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconPlus).Size(22, 22).TextColor(ui.Hex(colorMuted))
			})
			ui.Row(c.Key("actions")).Gap(8).Children(func() {
				accentAction(c.Key("add-text"), "添加文字", iconText, true, func() {
					e.add(item{Kind: kindText, X: 2, Y: 2})
				})
				outlineAction(c.Key("import"), "导入图片", iconPicture, true, func() {
					e.perform(func() error { return e.importImage(2, 2) })
				})
			})
		})
	})
}

// textToolbar 在选中的文字框上方显示格式和缩进快捷按钮
func (e *editor) textToolbar(c *ui.Context, a *item, paper ui.Rect) {
	state := e.textEdit
	if state == nil {
		return
	}
	doc := state.doc
	box := a.box(e.label, paper)
	top := box.Y - 48
	if top < 6 {
		top = box.Y + box.H + 10
	}
	left := max(6, box.X)
	t := c.Theme()
	ui.Row(c).Absolute().Left(left).Top(top).Height(38).Padding(0, 5).Gap(1).
		AlignItems(ui.Center).Radius(9).Background(ui.Hex(colorPanel)).
		BorderWidth(1).BorderColor(ui.Hex(colorFieldBorder)).
		Shadow(0, 10, 24, 0, ui.RGBA(16, 24, 40, 0.15)).Children(func() {
		formats := doc.Formatting()
		for _, style := range []struct {
			title     string
			glyph     string
			state     richtext.FormatState
			toggle    func()
			fontSize  float32
			weight    int
			italic    bool
			underline bool
		}{
			{"加粗", "B", formats.Bold, doc.ToggleBold, 12, 800, false, false},
			{"斜体", "I", formats.Italic, doc.ToggleItalic, 13, 400, true, false},
			{"下划线", "U", formats.Underline, doc.ToggleUnderline, 12, 600, false, true},
		} {
			style := style
			active := style.state == richtext.FormatOn
			button := ui.ButtonBase(c.Key(style.title)).Size(28, 28).Radius(6).Role(ui.RoleButton).
				Label(style.title).AccessSelected(active)
			ink := ui.Hex("#475467")
			if active {
				button.Background(ui.Hex(colorChip))
				ink = t.Accent
			} else if button.Hovered() {
				button.Background(ui.Hex(colorHover))
			}
			button.Children(func() {
				text := ui.Text(c, style.glyph).FontSize(style.fontSize).FontWeight(style.weight).TextColor(ink)
				if style.italic {
					text.Italic()
				}
				if style.underline {
					text.Underline()
				}
			})
			if button.Clicked() {
				formatText(doc, style.toggle)
				if state.writeTo(&a.labelText) {
					e.changed()
				}
			}
		}
		ui.Box(c.Key("rule")).Width(1).Height(18).Margin(4, 4).Shrink(0).Background(ui.Hex(colorDivider))
		for i, label := range [3]string{"左对齐", "水平居中", "右对齐"} {
			active := formats.Paragraph.Align == ui.Align(i)
			button := ui.ButtonBase(c.Key(label)).Size(28, 28).Radius(6).Role(ui.RoleButton).
				Label(label).AccessSelected(active)
			ink := ui.Hex("#475467")
			if active {
				button.Background(ui.Hex(colorChip))
				ink = t.Accent
			} else if button.Hovered() {
				button.Background(ui.Hex(colorHover))
			}
			align := ui.Align(i)
			button.Children(func() {
				ui.Icon(c, horizontalAlignIcons[i]).Size(14, 14).TextColor(ink)
			})
			if button.Clicked() {
				formatText(doc, func() { doc.SetAlign(align) })
				if state.writeTo(&a.labelText) {
					e.changed()
				}
			}
		}
	})
}

// zoomControl 在画布右下角显示缩放按钮和适合窗口
func (e *editor) zoomControl(c *ui.Context, viewport ui.Rect) {
	ui.Row(c).Absolute().Right(18).Bottom(18).Padding(4).Gap(2).Radius(10).
		AlignItems(ui.Center).Background(ui.Hex(colorPanel)).
		BorderWidth(1).BorderColor(ui.Hex(colorFieldBorder)).
		Shadow(0, 6, 20, 0, ui.RGBA(16, 24, 40, 0.10)).Children(func() {
		iconAction(c.Key("out"), "缩小", iconMinus, func() { e.zoomBy(1 / 1.25) })
		percent := ui.ButtonBase(c.Key("percent")).Height(28).Padding(0, 6).Radius(6).
			Role(ui.RoleButton).Label("缩放比例").Tooltip("适合窗口")
		if percent.Hovered() {
			percent.Background(ui.Hex(colorHover))
		}
		percent.Children(func() {
			ui.Textf(c, "%.0f%%", e.zoom*100).FontSize(11).FontWeight(600).TextColor(ui.Hex(colorBody))
		})
		if percent.Clicked() {
			e.fitView()
		}
		iconAction(c.Key("in"), "放大", iconPlus, func() { e.zoomBy(1.25) })
		ui.Box(c.Key("rule")).Width(1).Height(16).Margin(2, 2).Shrink(0).Background(ui.Hex(colorDivider))
		fit := ui.ButtonBase(c.Key("fit")).Height(28).Padding(0, 7).Gap(5).Radius(6).
			Role(ui.RoleButton).Label("适合窗口")
		if fit.Hovered() {
			fit.Background(ui.Hex(colorHover))
		}
		fit.Children(func() {
			ui.Icon(c, iconFit).Size(13, 13).TextColor(ui.Hex("#475467"))
			ui.Text(c, "适合窗口").FontSize(10.5).TextColor(ui.Hex("#475467"))
		})
		if fit.Clicked() {
			e.fitView()
		}
	})
}

// iconAction 显示缩放控件中的方形图标按钮
func iconAction(c *ui.Context, label string, icon *ui.SVG, action func()) {
	button := ui.ButtonBase(c).Size(28, 28).Radius(6).Role(ui.RoleButton).Label(label).Tooltip(label)
	if button.Hovered() {
		button.Background(ui.Hex(colorHover))
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(ui.Hex("#475467"))
	})
	if button.Clicked() {
		action()
	}
}

// paintCanvas 绘制点阵背景、纸张、标尺、选中框和对齐线
func (e *editor) paintCanvas(p *ui.Painter, viewport ui.Rect, t *ui.Theme) {
	p.Fill(viewport, ui.Hex(colorCanvas), 0)
	e.paintDots(p, viewport)
	p.Clip(viewport, 0, func() {
		paper := e.paperRect(viewport)
		p.Shadow(paper, 0, 0, 10, 30, -6, ui.RGBA(16, 24, 40, 0.18))
		p.Stroke(paper, ui.Hex(colorPaperBorder), 0, 1)
		e.label.paint(p, paper, false)
		e.paintRulers(p, viewport, paper)
		if a := e.selectedItem(); a != nil {
			box := a.selectionBox(e.label, paper)
			p.Stroke(box, t.Accent, 0, 1.5)
			if e.cells != nil && !e.inline {
				cells := a.rangeBox(box, e.cells.bounds(a))
				p.Fill(cells, t.Accent.Alpha(0.14), 0)
				p.Stroke(cells, t.Accent, 0, 1.5)
			}
			if !e.inline {
				for _, handle := range handleRects(box) {
					p.Fill(handle, ui.Hex(colorPanel), 2)
					p.Stroke(handle, t.Accent, 2, 1.5)
				}
			}
		}
		if d := e.drag; d != nil {
			color := ui.Hex(colorGuide)
			if d.guideX.visible {
				x := paper.X + float32(d.guideX.at/e.label.Width)*paper.W
				p.Line(x, paper.Y, x, paper.Y+paper.H, 1, color)
			}
			if d.guideY.visible {
				y := paper.Y + float32(d.guideY.at/e.label.Height)*paper.H
				p.Line(paper.X, y, paper.X+paper.W, y, 1, color)
			}
		}
	})
}

// paintDots 绘制画布背景的点阵
func (e *editor) paintDots(p *ui.Painter, viewport ui.Rect) {
	if !prefs.ShowGrid {
		return
	}
	color := ui.Hex(colorDot)
	for y := float32(canvasDotStep); y < viewport.H; y += canvasDotStep {
		for x := float32(canvasDotStep); x < viewport.W; x += canvasDotStep {
			p.Fill(ui.Rect{X: viewport.X + x, Y: viewport.Y + y, W: canvasDotSize, H: canvasDotSize}, color, 1)
		}
	}
}

// paintRulers 在纸张上方和左侧标注标签的毫米尺寸
func (e *editor) paintRulers(p *ui.Painter, viewport, paper ui.Rect) {
	line := ui.Hex(colorRuler)
	if top := paper.Y - 14; top > viewport.Y+8 {
		p.Line(paper.X, top, paper.X+paper.W, top, 1, line)
		p.Line(paper.X, top-3, paper.X, top+3, 1, line)
		p.Line(paper.X+paper.W, top-3, paper.X+paper.W, top+3, 1, line)
		layout := ui.ShapeText(fmt.Sprintf("%g mm", e.label.Width), ui.Font{Size: 10}, 0)
		w, h := layout.Size()
		x := paper.X + (paper.W-w)/2
		p.Fill(ui.Rect{X: x - 4, Y: top - h/2, W: w + 8, H: h}, ui.Hex(colorCanvas), 0)
		p.TextLayout(layout, x, top-h/2, ui.Hex(colorRulerText))
	}
	left := paper.X - 16
	if left <= viewport.X+8 {
		return
	}
	p.Line(left, paper.Y, left, paper.Y+paper.H, 1, line)
	p.Line(left-3, paper.Y, left+3, paper.Y, 1, line)
	p.Line(left-3, paper.Y+paper.H, left+3, paper.Y+paper.H, 1, line)
}

// heightRuler 在纸张左侧竖排显示标签高度
func (e *editor) heightRuler(c *ui.Context, paper, viewport ui.Rect) {
	left := paper.X - 16
	if left-40 < viewport.X {
		return
	}
	ui.Box(c).Absolute().Left(left-40).Top(paper.Y+paper.H/2-10).Size(60, 20).PassThrough().
		Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		text := ui.Textf(c.Key("text"), "%g mm", e.label.Height).FontSize(10).
			SingleLine().TextColor(ui.Hex(colorRulerText))
		text.Rotate(90).Background(ui.Hex(colorCanvas))
	})
}

// handleRects 返回选中框四角的手柄，顺序为左上、右上、左下、右下
func handleRects(box ui.Rect) [4]ui.Rect {
	size := float32(handleSize)
	return [4]ui.Rect{
		{X: box.X - size/2, Y: box.Y - size/2, W: size, H: size},
		{X: box.X + box.W - size/2, Y: box.Y - size/2, W: size, H: size},
		{X: box.X - size/2, Y: box.Y + box.H - size/2, W: size, H: size},
		{X: box.X + box.W - size/2, Y: box.Y + box.H - size/2, W: size, H: size},
	}
}

// selectionBox 计算选框并扩大细横线的鼠标命中区域
func (a *item) selectionBox(l *label, paper ui.Rect) ui.Rect {
	box := a.box(l, paper)
	if a.Kind == kindLine && box.H < 10 {
		box.Y -= (10 - box.H) / 2
		box.H = 10
	}
	return box
}

// scrollCanvas 平移画布或以鼠标位置为中心缩放标签
func (e *editor) scrollCanvas(viewport ui.Rect, ev ui.InputEvent) {
	dx, dy := ev.DX, ev.DY
	if ev.Mods&ui.Cmd != 0 {
		scale := ev.Scale
		if ev.Kind == ui.InputScroll {
			scale = float32(math.Exp(-float64(dy) / 500))
		}
		zoom := min(maxCanvasZoom, max(minCanvasZoom, e.zoom*scale))
		change := zoom/e.zoom - 1
		dx = (ev.X - viewport.W*(0.5+e.pan.X)) * change
		dy = (ev.Y - viewport.H*(0.5+e.pan.Y)) * change
		e.zoom = zoom
	} else if ev.Mods&ui.Shift != 0 {
		if dx == 0 {
			dx = dy
		}
		dy = 0
	}
	e.pan.X = min(0.5, max(-0.5, e.pan.X-dx/viewport.W))
	e.pan.Y = min(0.5, max(-0.5, e.pan.Y-dy/viewport.H))
}

// handleCanvas 处理画布滚轮、元素和单元格选择、编辑及拖动
func (e *editor) handleCanvas(c *ui.Context, ev ui.InputEvent) bool {
	bounds := e.canvas.Bounds(c)
	viewport := ui.Rect{W: bounds.W, H: bounds.H}
	paper := e.paperRect(viewport)
	l := e.label
	switch ev.Kind {
	case ui.InputScroll, ui.InputZoom:
		if ev.Kind == ui.InputZoom && ev.Mods&ui.Cmd == 0 {
			return false
		}
		if e.drag == nil {
			e.scrollCanvas(viewport, ev)
		}
		return true
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		if a := e.selectedItem(); a != nil && !e.inline {
			for i, handle := range handleRects(a.selectionBox(l, paper)) {
				if handle.Contains(ev.X, ev.Y) {
					e.cells = nil
					e.drag = &gesture{item: *a, startX: ev.X, startY: ev.Y, resize: true, corner: i}
					return true
				}
			}
		}
		previous := e.selected
		for i := len(l.Items) - 1; i >= 0; i-- {
			a := &l.Items[i]
			box := a.selectionBox(l, paper)
			if box.Contains(ev.X, ev.Y) {
				e.selectItem(i)
				if a.Kind == kindTable && tableInterior(box).Contains(ev.X, ev.Y) {
					e.selectTableCells(a, box, ev)
					return true
				}
				e.cells = nil
				e.inline = ev.Clicks == 2 && i == previous && a.Kind == kindText
				if !e.inline {
					e.drag = &gesture{item: *a, startX: ev.X, startY: ev.Y}
				}
				return true
			}
		}
		e.selectItem(-1)
		return true
	case ui.InputPointerMove:
		if e.drag == nil {
			return false
		}
		d := e.drag
		a := &l.Items[e.selected]
		if d.selectCells {
			e.cells.end = a.cellAt(a.box(l, paper), ev.X, ev.Y)
			return true
		}
		dx := float64(ev.X-d.startX) * l.Width / float64(paper.W)
		dy := float64(ev.Y-d.startY) * l.Height / float64(paper.H)
		if d.resize {
			e.resize(d, a, dx, dy)
		} else if !prefs.SnapGuides {
			a.X, a.Y = d.item.X+dx, d.item.Y+dy
			d.guideX, d.guideY = alignmentGuide{}, alignmentGuide{}
		} else {
			a.X, d.guideX = e.alignDrag(d.item.X+dx, a.W, l.Width, alignmentDistance*l.Width/float64(paper.W), true)
			a.Y, d.guideY = e.alignDrag(d.item.Y+dy, a.H, l.Height, alignmentDistance*l.Height/float64(paper.H), false)
		}
		a.constrain(l)
		return true
	case ui.InputPointerUp:
		if e.drag == nil {
			return false
		}
		d := e.drag
		a := &l.Items[e.selected]
		e.drag = nil
		if d.selectCells {
			if ev.Canceled {
				e.cells = d.previousCells
			}
			return true
		}
		before := d.item
		if ev.Canceled {
			*a = before
		} else if before.X != a.X || before.Y != a.Y || before.W != a.W || before.H != a.H {
			e.changed()
		}
		return true
	case ui.InputKeyDown:
		if (ev.Key == ui.KeyDelete || ev.Key == ui.KeyBackspace) && e.selected >= 0 {
			if e.cells == nil {
				e.remove()
			}
			return true
		}
	}
	return false
}

// resize 按拖动的角调整元素尺寸，左上和上边分别向左上扩展
func (e *editor) resize(d *gesture, a *item, dx, dy float64) {
	left, top := d.corner%2 == 0, d.corner < 2
	width, height := d.item.W+dx, d.item.H+dy
	if left {
		width = d.item.W - dx
	}
	if top {
		height = d.item.H - dy
	}
	width = min(e.label.Width, max(minimumItem, roundMM(width)))
	height = min(e.label.Height, max(a.minimumHeight(), roundMM(height)))
	if left {
		a.X = roundMM(d.item.X + d.item.W - width)
	}
	if top {
		a.Y = roundMM(d.item.Y + d.item.H - height)
	}
	a.W, a.H = width, height
}

// alignDrag 计算单轴上最近的边缘或中心对齐位置
func (e *editor) alignDrag(start, size, limit, distance float64, horizontal bool) (float64, alignmentGuide) {
	position, guide := roundMM(start), alignmentGuide{}
	compare := func(origin, span float64) {
		for _, target := range [3]float64{origin, origin + span/2, origin + span} {
			for _, offset := range [3]float64{0, size / 2, size} {
				next := target - offset
				delta := math.Abs(next - start)
				if within(next, 0, limit-size) && delta <= distance && (!guide.visible || delta < distance) {
					position, distance = next, delta
					guide = alignmentGuide{at: target, visible: true}
				}
			}
		}
	}
	compare(0, limit)
	for i, other := range e.label.Items {
		if i != e.selected {
			if horizontal {
				compare(other.X, other.W)
			} else {
				compare(other.Y, other.H)
			}
		}
	}
	return position, guide
}

// roundMM 将未吸附的拖动结果舍入到 0.1 毫米
func roundMM(value float64) float64 {
	return math.Round(value*10) / 10
}

// add 添加带默认尺寸的元素并选中它
func (e *editor) add(a item) {
	switch a.Kind {
	case kindText:
		a.W, a.H, a.FontSize, a.Value = 24, 6, 4, "标签文字"
	case kindQR:
		a.W, a.H, a.Value = 18, 18, "GoLabel"
	case kindBarcode:
		a.W, a.H, a.Value = 32, 9, "12345678"
	case kindImage:
		a.W, a.H = 18, 18
	case kindLine:
		a.W, a.H = 28, 0.3
	case kindTable:
		a.W, a.H, a.Rows, a.Columns, a.Stroke = 30, 16, 3, 3, 0.3
	}
	if a.Kind == kindQR || a.Kind == kindBarcode {
		a.prepareSymbol(e.graphics)
	}
	a.constrain(e.label)
	e.label.Items = append(e.label.Items, a)
	e.selectItem(len(e.label.Items) - 1)
	e.changed()
	e.status = ""
}

// remove 删除当前选中的标签元素
func (e *editor) remove() {
	e.label.Items = slices.Delete(e.label.Items, e.selected, e.selected+1)
	e.selected, e.drag = -1, nil
	e.textEdit, e.cells, e.inline = nil, nil, false
	e.changed()
}

// perform 将文件操作和系统对话框排入窗口事件队列
func (e *editor) perform(action func() error) {
	e.window.Update(func() {
		if err := action(); err != nil && !errors.Is(err, mygo.ErrCanceled) {
			e.status = err.Error()
		}
	})
}

// importImage 打开图片并将原图内嵌到标签的指定位置
func (e *editor) importImage(x, y float64) error {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: e.window, Title: "添加图片",
		Filters: []mygo.FileFilter{{Name: "PNG 或 JPEG", Extensions: []string{"png", "jpg", "jpeg"}}},
	})
	if err != nil || len(paths) == 0 {
		return err
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return err
	}
	bitmap, err := decodeImage(data)
	if err != nil {
		return err
	}
	e.add(item{Kind: kindImage, X: x, Y: y, Data: data, bitmap: bitmap})
	return nil
}

// exportPDF 将当前标签保存为实际尺寸的 PDF
func (e *editor) exportPDF() error {
	data, err := e.graphics.pdf(e.label)
	if err != nil {
		return err
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: e.window, Title: "导出标签 PDF", DefaultPath: "标签.pdf",
		Filters: []mygo.FileFilter{{Name: "PDF", Extensions: []string{"pdf"}}},
	})
	if err != nil || path == "" {
		return err
	}
	if err := writePDFFile(path, data); err != nil {
		return err
	}
	e.status = "已导出 PDF"
	return nil
}

// print 打开系统打印面板并显示提交结果
func (e *editor) print() error {
	sent, err := e.graphics.print(e.label, e.documentName(), e.window.NativeHandle())
	if sent {
		e.status = "已提交打印任务"
	}
	return err
}

// paperRect 计算标签在给定画布内的大小和位置
func (e *editor) paperRect(viewport ui.Rect) ui.Rect {
	w, h := float32(e.label.Width), float32(e.label.Height)
	fit := min(max(1, viewport.W-108)/w, max(1, viewport.H-100)/h, 12)
	width, height := w*fit*e.zoom, h*fit*e.zoom
	return ui.Rect{
		X: viewport.X + (viewport.W-width)/2 + e.pan.X*viewport.W,
		Y: viewport.Y + (viewport.H-height)/2 + e.pan.Y*viewport.H,
		W: width,
		H: height,
	}
}

// numberDraft 保存尺寸输入中的文字和已提交数值
type numberDraft struct {
	text  string
	value float64
}

// inspectorDimension 显示属性栏中的一个毫米尺寸字段
func inspectorDimension(c *ui.Context, name string, value *float64, low, high float64) {
	field := c.Key(name)
	decimals := 1
	if v := *value; v == math.Trunc(v) {
		decimals = 0
	}
	ui.Column(field).Grow(1).MinWidth(0).Gap(5).Children(func() {
		ui.Text(field, name).FontSize(10).TextColor(ui.Hex(colorMuted))
		inspectorNumber(field, name, "mm", 0, value, low, high, decimals, false)
	})
}

// inspectorNumber 显示指定精度和单位的紧凑步进输入框
func inspectorNumber(c *ui.Context, name, unit string, width float32, value *float64, low, high float64, decimals int, stepper bool) {
	high = max(low, high)
	scale := math.Pow10(decimals)
	step := 1 / scale
	format := func(value float64) string { return strconv.FormatFloat(value, 'f', decimals, 64) }
	label := name
	if unit != "" {
		label += "（" + unit + "）"
	}
	row := ui.Row(c.Key("row")).Height(34).MinWidth(0).AlignItems(ui.Center).Radius(7).Clip().
		Border(1, ui.Hex(colorFieldBorder)).Background(ui.Hex(colorPanel))
	if width > 0 {
		row.Width(width)
	}
	draft := ui.Local(row, "number", func() numberDraft {
		return numberDraft{text: format(*value), value: *value}
	})
	if draft.value != *value {
		draft.value, draft.text = *value, format(*value)
	}
	row.Children(func() {
		in := ui.TextInputBase(c.Key("entry"), &draft.text).Grow(1).MinWidth(0).Height(32).
			Padding(0, 10).FontSize(12).Label(label)
		if in.Changed() {
			if n, err := strconv.ParseFloat(draft.text, 64); err == nil && within(n, low, high) {
				*value = math.Round(n*scale) / scale
			}
		}
		delta := 0.0
		switch {
		case in.Shortcut(0, ui.KeyUp):
			delta = step
		case in.Shortcut(0, ui.KeyDown):
			delta = -step
		}
		if delta != 0 {
			*value = min(high, max(low, math.Round((*value+delta)*scale)/scale))
		}
		if unit != "" {
			ui.Text(c, unit).FontSize(10).TextColor(ui.Hex(colorFaint)).Margin(0, 4, 0, 0)
		}
		stepped := false
		if stepper {
			stepped = ui.Stepper(c.Key("stepper"), value, low, high, step).Width(20).Height(32).Radius(0).
				BorderWidth(0, 0, 0, 1).BorderColor(ui.Hex(colorFieldBorder)).
				Label(name + "步进").Changed()
		}
		if !in.Focused() || stepped || delta != 0 {
			draft.text = format(*value)
		}
		draft.value = *value
	})
}
