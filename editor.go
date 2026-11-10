package main

import (
	"errors"
	"math"
	"os"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	// minCanvasZoom 是画布允许的最小缩放比例
	minCanvasZoom = 0.5
	// maxCanvasZoom 是画布允许的最大缩放比例
	maxCanvasZoom = 2
	// alignmentDistance 是触发对齐吸附的屏幕距离
	alignmentDistance = 5
	// canvasGridStep 是画布小方格的屏幕边长
	canvasGridStep = 10
	// canvasGridGroup 是一个大方格边长包含的小格数
	canvasGridGroup = 4
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
}

// gesture 记录元素拖动或单元格拖选的起点状态
type gesture struct {
	item           item
	startX, startY float32
	resize         bool
	guideX, guideY alignmentGuide
	selectCells    bool
	previousCells  *tableSelection
}

// alignmentGuide 保存标签毫米坐标中的一条对齐线
type alignmentGuide struct {
	at      float64
	visible bool
}

// view 构建工具栏、画布、属性栏和状态栏
func (e *editor) view(c *ui.Context) {
	l := e.document.Content()
	if l != e.label {
		e.label, e.selected, e.drag = l, -1, nil
		e.textEdit, e.cells, e.inline = nil, nil, false
		for i := range l.Items {
			element := &l.Items[i]
			if element.Kind == kindQR || element.Kind == kindBarcode {
				element.prepareSymbol(e.graphics)
			}
		}
	}

	t := c.Theme()
	c.OnShortcut(ui.Cmd, ui.KeyP, func() { e.perform(e.print) })
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		e.toolbar(c.Key("toolbar"))
		ui.Row(c.Key("workspace")).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			e.tools(c.Key("tools"))
			e.canvasView(c.Key("canvas"))
			e.inspector(c.Key("inspector"))
		})
		ui.Row(c.Key("status")).Height(32).Padding(0, 14).AlignItems(ui.Center).Gap(12).BorderWidth(1, 0, 0).BorderColor(t.Border).Children(func() {
			e.paperSettings(c.Key("paper-settings"))
			ui.Box(c).Grow(1)
			ui.Text(c, e.status).FontSize(12).TextColor(t.TextMuted)
			ui.Textf(c, "%d 个元素", len(l.Items)).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

// toolbar 提供模板、PDF 和打印操作
func (e *editor) toolbar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(57).Padding(0, 14).Gap(7).AlignItems(ui.Center).BorderWidth(0, 0, 1).BorderColor(t.Border).Children(func() {
		ui.Text(c, "GoLabel").FontSize(18).Bold().Margin(0, 10, 0, 0)
		if ui.Button(c.Key("new"), "新建").Clicked() {
			e.perform(func() error { e.documents.New(); return nil })
		}
		if ui.Button(c.Key("open"), "打开").Clicked() {
			e.perform(func() error {
				_, err := e.documents.OpenDialog()
				return err
			})
		}
		if ui.Button(c.Key("save"), "保存").Clicked() {
			e.perform(e.document.Save)
		}
		ui.Box(c).Grow(1)
		empty := len(e.label.Items) == 0
		if ui.Button(c.Key("pdf"), "导出 PDF").Disabled(empty).Clicked() {
			e.perform(e.exportPDF)
		}
		if ui.PrimaryButton(c.Key("print"), "打印…").Disabled(empty).Clicked() {
			e.perform(e.print)
		}
	})
}

// tools 显示可以点击或拖入标签的元素工具
func (e *editor) tools(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(123).FillHeight().Shrink(0).Padding(16, 12).Gap(9).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Text(c, "添加元素").FontSize(12).Bold().TextColor(t.TextMuted)
		for _, entry := range []struct {
			kind itemKind
			name string
		}{
			{kindText, "文字"}, {kindImage, "图片…"}, {kindBarcode, "Code 128"}, {kindQR, "二维码"},
			{kindLine, "横线"}, {kindTable, "表格"},
		} {
			button := ui.Button(c.Key(entry.kind), entry.name).FillWidth().Drag(entry.kind)
			if x, y, ok := e.toolPosition(c, button); ok {
				if entry.kind == kindImage {
					e.perform(func() error { return e.importImage(x, y) })
				} else {
					e.add(item{Kind: entry.kind, X: x, Y: y})
				}
			}
		}
	})
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

// paperSettings 以状态栏规格文字打开纸张尺寸和 DPI 设置
func (e *editor) paperSettings(c *ui.Context) {
	l := e.label
	button := ui.ButtonBase(c.Key("summary")).Height(24).Padding(0, 8).Radius(5).
		BorderWidth(1).Cursor(ui.CursorPointer).Role(ui.RoleButton)
	if button.Hovered() {
		button.BorderColor(c.Theme().Border)
	}
	open := ui.Local(button, "open", func() bool { return false })
	if button.Clicked() {
		*open = !*open
	}
	ui.Popover(c.Key("popover"), button, open, func() {
		ui.Column(c.Key("fields")).Width(218).Padding(10).Gap(8).Children(func() {
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
			ui.Row(c.Key("dpi-row")).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "打印 DPI").FontSize(11).TextColor(c.Theme().TextMuted)
				if ui.Segmented(c.Key("dpi"), &selectedDPI, "203", "300", "600").Grow(1).Label("打印分辨率").Changed() {
					l.DPI = resolutions[selectedDPI]
				}
			})
			if w != l.Width || h != l.Height || dpi != l.DPI {
				e.document.Changed()
			}
		})
	})
	button.Children(func() {
		ui.Textf(c, "%g × %g mm  ·  %d DPI", l.Width, l.Height, l.DPI).FontSize(12).SingleLine()
	})
}

// inspector 显示所选元素的格式及几何属性
func (e *editor) inspector(c *ui.Context) {
	t, l := c.Theme(), e.label
	ui.Scroll(c).Width(236).FillHeight().Shrink(0).BorderWidth(0, 0, 0, 1).BorderColor(t.Border).Children(func() {
		ui.Column(c.Key("properties")).FillWidth().Padding(14).Gap(7).Children(func() {
			if e.selected < 0 {
				ui.Text(c, "元素属性").FontSize(13).Bold()
				ui.Text(c, "选择画布中的元素").FontSize(12).TextColor(t.TextMuted)
				return
			}

			a := &l.Items[e.selected]
			fields := c.Key(e.selected)
			title := "元素属性"
			switch a.Kind {
			case kindText:
				title = "文字"
			case kindImage:
				title = "图片"
			case kindBarcode:
				title = "条码"
			case kindQR:
				title = "二维码"
			case kindLine:
				title = "横线"
			case kindTable:
				title = "表格"
			}
			remove := false
			ui.Row(fields.Key("heading")).AlignItems(ui.Center).Children(func() {
				ui.Text(fields, title).FontSize(13).Bold().Grow(1)
				button := ui.ButtonBase(fields.Key("delete")).Size(24, 24).Radius(5).Role(ui.RoleButton).
					Label("删除元素").Tooltip("删除元素")
				color := t.TextMuted
				if button.Hovered() {
					button.Background(t.Danger.Alpha(0.10))
					color = t.Danger
				}
				button.Children(func() { ui.Icon(fields, removeElementIcon).Size(14, 14).TextColor(color) })
				remove = button.Clicked()
			})
			if remove {
				e.remove()
				return
			}
			switch a.Kind {
			case kindText:
				if e.textProperties(fields, &a.labelText) {
					e.document.Changed()
				}
			case kindQR, kindBarcode:
				if ui.TextInput(fields.Key("value"), &a.Value).Placeholder("输入内容").Label("编码内容").FillWidth().Changed() {
					a.prepareSymbol(e.graphics)
					e.document.Changed()
				}
			case kindTable:
				e.tableProperties(fields, a)
			}
			if a.issue != nil {
				ui.Text(fields, a.issue.Error()).FontSize(11).TextColor(t.Danger).Wrap()
			}

			ui.Box(fields.Key("geometry-divider")).Height(1).FillWidth().Background(t.Border).Margin(3, 0)
			ui.Text(c, "位置与尺寸").FontSize(13).Bold()
			x, y, aw, ah := a.X, a.Y, a.W, a.H
			ui.Row(fields.Key("position")).Gap(8).Children(func() {
				inspectorDimension(fields, "X", &a.X, 0, l.Width-a.W)
				inspectorDimension(fields, "Y", &a.Y, 0, l.Height-a.H)
			})
			ui.Row(fields.Key("size")).Gap(8).Children(func() {
				width, height := "宽度", "高度"
				if a.Kind == kindLine {
					width, height = "长度", "线宽"
				}
				inspectorDimension(fields, width, &a.W, minimumItem, l.Width-a.X)
				inspectorDimension(fields, height, &a.H, a.minimumHeight(), l.Height-a.Y)
			})
			if x != a.X || y != a.Y || aw != a.W || ah != a.H {
				a.constrain(l)
				e.document.Changed()
			}
		})
	})
}

// numberDraft 保存尺寸输入中的文字和已提交数值
type numberDraft struct {
	text  string
	value float64
}

// inspectorDimension 显示属性栏中的一个毫米尺寸字段
func inspectorDimension(c *ui.Context, name string, value *float64, low, high float64) {
	field := c.Key(name)
	ui.Column(field).Grow(1).MinWidth(0).Gap(4).Children(func() {
		ui.Text(field, name).FontSize(11).TextColor(c.Theme().TextMuted)
		inspectorNumber(field, name, "mm", 0, value, low, high, 1)
	})
}

// inspectorNumber 显示指定精度和单位的紧凑步进输入框
func inspectorNumber(c *ui.Context, name, unit string, width float32, value *float64, low, high float64, decimals int) {
	t := c.Theme()
	high = max(low, high)
	scale := math.Pow10(decimals)
	step := 1 / scale
	format := func(value float64) string { return strconv.FormatFloat(value, 'f', decimals, 64) }
	label := name
	if unit != "" {
		label += " (" + unit + ")"
	}
	row := ui.Row(c).Height(28).MinWidth(0).AlignItems(ui.Center).Radius(6).Border(1, t.Border).Background(t.Background).Clip()
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
		in := ui.TextInputBase(c.Key("entry"), &draft.text).Grow(1).MinWidth(0).Height(26).Padding(0, 4).
			FontSize(12).TextAlign(ui.Center).Label(label)
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
			ui.Text(c, unit).FontSize(10).TextColor(t.TextMuted).Margin(0, 4, 0, 0)
		}
		stepped := ui.Stepper(c.Key("stepper"), value, low, high, step).Width(16).Height(26).Radius(0).
			BorderWidth(0, 0, 0, 1).BorderColor(t.Border).Label(name + "步进").Changed()
		if !in.Focused() || stepped || delta != 0 {
			draft.text = format(*value)
		}
		draft.value = *value
	})
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

// resizeHandle 计算元素右下角的缩放手柄
func resizeHandle(box ui.Rect) ui.Rect {
	return ui.Rect{X: box.X + box.W - 5, Y: box.Y + box.H - 5, W: 10, H: 10}
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

// canvasView 构建画布、原位编辑器和缩放读数
func (e *editor) canvasView(c *ui.Context) {
	t := c.Theme()
	canvas := ui.Box(c).Grow(1).MinWidth(0).FillHeight().Clip().Focusable().Label("标签画布").
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
		ui.Textf(c.Key("zoom"), "%.0f%%", e.zoom*100).Absolute().Left(10).Bottom(10).
			Padding(3, 7).Radius(5).Background(t.Background).FontSize(12).TextColor(t.TextMuted).PassThrough()
	})
}

// paintCanvas 在工作区内绘制大小方格、标签、选中框和对齐线
func (e *editor) paintCanvas(p *ui.Painter, viewport ui.Rect, t *ui.Theme) {
	p.Fill(viewport, t.Background, 0)
	minor := t.Border.Alpha(0.3)
	for i := 1; ; i++ {
		offset := float32(i * canvasGridStep)
		if offset >= max(viewport.W, viewport.H) {
			break
		}
		color := minor
		if i%canvasGridGroup == 0 {
			color = t.Border
		}
		if offset < viewport.W {
			x := viewport.X + offset
			p.Line(x, viewport.Y, x, viewport.Y+viewport.H, 1, color)
		}
		if offset < viewport.H {
			y := viewport.Y + offset
			p.Line(viewport.X, y, viewport.X+viewport.W, y, 1, color)
		}
	}
	p.Clip(viewport, 0, func() {
		paper := e.paperRect(viewport)
		p.Shadow(paper, 0, 0, 5, 14, 0, ui.Hex("#00000026"))
		e.label.paint(p, paper, false)
		if e.selected >= 0 {
			a := &e.label.Items[e.selected]
			box := a.selectionBox(e.label, paper)
			p.Stroke(box, t.Accent, 0, 1.5)
			if e.cells != nil && !e.inline {
				cells := a.rangeBox(box, e.cells.bounds(a))
				p.Fill(cells, t.Accent.Alpha(0.14), 0)
				p.Stroke(cells, t.Accent, 0, 1.5)
			}
			if !e.inline {
				p.Fill(resizeHandle(box), t.Accent, 2)
			}
		}
		if d := e.drag; d != nil {
			color := ui.Hex("#E33678")
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
		if e.selected >= 0 && !e.inline {
			a := &l.Items[e.selected]
			if resizeHandle(a.selectionBox(l, paper)).Contains(ev.X, ev.Y) {
				e.cells = nil
				e.drag = &gesture{item: *a, startX: ev.X, startY: ev.Y, resize: true}
				return true
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
			a.W = min(l.Width-a.X, roundMM(d.item.W+dx))
			a.H = min(l.Height-a.Y, roundMM(d.item.H+dy))
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
			e.document.Changed()
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
	e.document.Changed()
	e.status = ""
}

// remove 删除当前选中的标签元素
func (e *editor) remove() {
	e.label.Items = slices.Delete(e.label.Items, e.selected, e.selected+1)
	e.selected, e.drag = -1, nil
	e.textEdit, e.cells, e.inline = nil, nil, false
	e.document.Changed()
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
	sent, err := e.graphics.print(e.label, e.document.Name(), e.window.NativeHandle())
	if sent {
		e.status = "已提交打印任务"
	}
	return err
}
