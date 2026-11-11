package editor

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/pengh/GoLabel/internal/i18n"
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
	// handleSize 是选中框四角手柄的屏幕边长
	handleSize = 8
)

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
		canvas := ui.Box(c.Key("surface")).Fill().Clip().Focusable().Label(i18n.Tr("canvas.label")).
			Bind(&e.canvas).HandleInput(func(ev ui.InputEvent) bool { return e.handleCanvas(c, ev) }).
			Draw(func(p *ui.Painter, viewport ui.Rect) { e.paintCanvas(p, viewport, t) })
		if e.inline && (canvas.Shortcut(0, ui.KeyEscape) ||
			(e.label.Items[e.selected].Kind == kindTable && canvas.Shortcut(0, ui.KeyEnter))) {
			e.inline = false
			canvas.Focus()
		}
		if e.inline && (e.tabs == nil || e.tabs.active == e) && canvas.FocusLeft() {
			e.inline = false
		}
		canvas.Children(func() {
			if e.inline && (e.tabs == nil || e.tabs.active == e) {
				e.inlineEditor(c.Key("inline-text"), canvas)
			}
		})

		bounds := canvas.Bounds()
		viewport := ui.Rect{W: bounds.W, H: bounds.H}
		e.trackViewport(c, canvas, viewport)
		paper := e.paperRect(viewport)
		if a := e.selectedItem(); a != nil && a.Kind == kindText && !e.inline {
			e.textToolbar(c.Key("text-toolbar"), a, paper)
		}
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
	ui.Row(c).Absolute().Left(left).Top(top).Height(38).Padding(0, 5).Gap(1).
		AlignItems(ui.Center).Radius(9).Background(colors.Floating).
		BorderWidth(1).BorderColor(colors.FieldBorder).
		Shadow(0, 10, 24, 0, colors.ShadowMid).Children(func() {
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
			{i18n.Tr("text.bold"), "B", formats.Bold, doc.ToggleBold, 12, 800, false, false},
			{i18n.Tr("text.italic"), "I", formats.Italic, doc.ToggleItalic, 13, 400, true, false},
			{i18n.Tr("text.underline"), "U", formats.Underline, doc.ToggleUnderline, 12, 600, false, true},
		} {
			style := style
			active := style.state == richtext.FormatOn
			button := ui.ButtonBase(c.Key(style.title)).Size(28, 28).Radius(6).Role(ui.RoleButton).
				Label(style.title).AccessSelected(active)
			ink := colors.Body2
			if active {
				button.Background(colors.ChipSelected)
				ink = colors.Accent
			} else if button.Hovered() {
				button.Background(colors.Hover)
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
		ui.Box(c.Key("rule")).Width(1).Height(18).Margin(4, 4).Shrink(0).Background(colors.Divider)
		for i, label := range [3]string{i18n.Tr("text.alignLeft"), i18n.Tr("text.alignCenter"), i18n.Tr("text.alignRight")} {
			active := formats.Paragraph.Align == ui.Align(i)
			button := ui.ButtonBase(c.Key(label)).Size(28, 28).Radius(6).Role(ui.RoleButton).
				Label(label).AccessSelected(active)
			ink := colors.Body2
			if active {
				button.Background(colors.ChipSelected)
				ink = colors.Accent
			} else if button.Hovered() {
				button.Background(colors.Hover)
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
		AlignItems(ui.Center).Background(colors.Floating).
		BorderWidth(1).BorderColor(colors.FieldBorder).
		Shadow(0, 6, 20, 0, colors.ShadowSubtle).Children(func() {
		iconAction(c.Key("out"), i18n.Tr("canvas.zoomOut"), iconMinus, func() { e.zoomBy(1 / 1.25) })
		percent := ui.ButtonBase(c.Key("percent")).Height(28).Padding(0, 6).Radius(6).
			Role(ui.RoleButton).Label(i18n.Tr("canvas.zoomLevel")).Tooltip(i18n.Tr("canvas.fit"))
		if percent.Hovered() {
			percent.Background(colors.Hover)
		}
		percent.Children(func() {
			ui.Textf(c, "%.0f%%", e.zoom*100).FontSize(11).FontWeight(600).TextColor(colors.Body)
		})
		if percent.Clicked() {
			e.fitView()
		}
		iconAction(c.Key("in"), i18n.Tr("canvas.zoomIn"), iconPlus, func() { e.zoomBy(1.25) })
		ui.Box(c.Key("rule")).Width(1).Height(16).Margin(2, 2).Shrink(0).Background(colors.Divider)
		fit := ui.ButtonBase(c.Key("fit")).Height(28).Padding(0, 7).Gap(5).Radius(6).
			Role(ui.RoleButton).Label(i18n.Tr("canvas.fit"))
		if fit.Hovered() {
			fit.Background(colors.Hover)
		}
		fit.Children(func() {
			ui.Icon(c, iconFit).Size(13, 13).TextColor(colors.Body2)
			ui.Text(c, i18n.Tr("canvas.fit")).FontSize(10.5).TextColor(colors.Body2)
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
		button.Background(colors.Hover)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(colors.Body2)
	})
	if button.Clicked() {
		action()
	}
}

// paintCanvas 绘制点阵背景、纸张、标尺、选中框和对齐线
func (e *editor) paintCanvas(p *ui.Painter, viewport ui.Rect, t *ui.Theme) {
	p.Fill(viewport, colors.Canvas, 0)
	e.paintDots(p, viewport)
	p.Clip(viewport, 0, func() {
		paper := e.paperRect(viewport)
		p.Shadow(paper, 0, 0, 10, 30, -6, colors.ShadowStrong)
		p.Stroke(paper, colors.PaperBorder, 0, 1)
		e.label.paint(p, paper, false)
		e.paintRulers(p, viewport, paper)
		if a := e.selectedItem(); a != nil {
			box := a.selectionBox(e.label, paper)
			p.Stroke(box, colors.Accent, 0, 1.5)
			if e.cells != nil && !e.inline {
				cells := a.rangeBox(box, e.cells.bounds(a))
				p.Fill(cells, colors.Accent.Alpha(0.14), 0)
				p.Stroke(cells, colors.Accent, 0, 1.5)
			}
			if !e.inline {
				for _, handle := range handleRects(box) {
					p.Fill(handle, colors.Paper, 2)
					p.Stroke(handle, colors.Accent, 2, 1.5)
				}
			}
		}
		if d := e.drag; d != nil {
			color := colors.Guide
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
	color := colors.Dot
	for y := float32(canvasDotStep); y < viewport.H; y += canvasDotStep {
		for x := float32(canvasDotStep); x < viewport.W; x += canvasDotStep {
			p.Fill(ui.Rect{X: viewport.X + x, Y: viewport.Y + y, W: canvasDotSize, H: canvasDotSize}, color, 1)
		}
	}
}

// paintRulers 在纸张上方和左侧标注标签的毫米尺寸
func (e *editor) paintRulers(p *ui.Painter, viewport, paper ui.Rect) {
	line := colors.Ruler
	if top := paper.Y - 14; top > viewport.Y+8 {
		p.Line(paper.X, top, paper.X+paper.W, top, 1, line)
		p.Line(paper.X, top-3, paper.X, top+3, 1, line)
		p.Line(paper.X+paper.W, top-3, paper.X+paper.W, top+3, 1, line)
		layout := ui.ShapeText(fmt.Sprintf("%g mm", e.label.Width), ui.Font{Size: 10}, 0)
		w, h := layout.Size()
		x := paper.X + (paper.W-w)/2
		p.Fill(ui.Rect{X: x - 4, Y: top - h/2, W: w + 8, H: h}, colors.Canvas, 0)
		p.TextLayout(layout, x, top-h/2, colors.RulerText)
	}
	left := paper.X - 16
	if left <= viewport.X+8 {
		return
	}
	p.Line(left, paper.Y, left, paper.Y+paper.H, 1, line)
	p.Line(left-3, paper.Y, left+3, paper.Y, 1, line)
	p.Line(left-3, paper.Y+paper.H, left+3, paper.Y+paper.H, 1, line)
	layout := ui.ShapeText(fmt.Sprintf("%g mm", e.label.Height), ui.Font{Size: 10}, 0)
	w, h := layout.Size()
	centerX, centerY := left-10, paper.Y+paper.H/2
	if centerX-h/2 < viewport.X {
		return
	}
	p.Transform(ui.Rotation(90), centerX, centerY, func() {
		x, y := centerX-w/2, centerY-h/2
		p.Fill(ui.Rect{X: x, Y: y, W: w, H: h}, colors.Canvas, 0)
		p.TextLayout(layout, x, y, colors.RulerText)
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
