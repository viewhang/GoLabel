package editor

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

// lineStyle 区分实线、虚线、点线、双线和箭头
type lineStyle string

const (
	// lineSolid 表示普通实线并作为旧文件的默认线型
	lineSolid lineStyle = ""
	// lineDashed 表示等间距虚线
	lineDashed lineStyle = "dashed"
	// lineDotted 表示等间距圆点线
	lineDotted lineStyle = "dotted"
	// lineDouble 表示两条平行实线
	lineDouble lineStyle = "double"
	// lineArrow 表示向右的单向箭头
	lineArrow lineStyle = "arrow"
	// lineDoubleArrow 表示两端均带箭头的线条
	lineDoubleArrow lineStyle = "double-arrow"
)

// lineStyles 保存线型选择框的选项顺序
var lineStyles = [...]lineStyle{lineSolid, lineDashed, lineDotted, lineDouble, lineArrow, lineDoubleArrow}

// lineWidth 返回毫米线宽并兼容以高度保存线宽的旧横线
func (a *item) lineWidth() float64 {
	if a.Stroke == 0 {
		return a.H
	}
	return a.Stroke
}

// validLine 校验线型和独立线宽并保留旧横线的尺寸格式
func (a *item) validLine() bool {
	return slices.Contains(lineStyles[:], a.LineStyle) &&
		((a.Stroke == 0 && a.LineStyle == lineSolid) || within(a.Stroke, minimumLineWidth, min(a.W, a.H)))
}

// validShape 校验图形的线宽和矩形圆角半径
func (a *item) validShape() bool {
	return within(a.Stroke, minimumLineWidth, min(a.W, a.H)/2) &&
		within(a.CornerRadius, 0, min(a.W, a.H)/2)
}

// shapePath 在给定区域内构建椭圆、菱形、三角形、五角星或六边形
func shapePath(kind itemKind, r ui.Rect) *ui.Path {
	path := new(ui.Path)
	if kind == kindEllipse {
		// k 是四分之一椭圆贝塞尔曲线的控制点比例
		const k = float32(0.5522847498)
		x, y, rx, ry := r.X+r.W/2, r.Y+r.H/2, r.W/2, r.H/2
		return path.MoveTo(x+rx, y).
			CubeTo(x+rx, y+k*ry, x+k*rx, y+ry, x, y+ry).
			CubeTo(x-k*rx, y+ry, x-rx, y+k*ry, x-rx, y).
			CubeTo(x-rx, y-k*ry, x-k*rx, y-ry, x, y-ry).
			CubeTo(x+k*rx, y-ry, x+rx, y-k*ry, x+rx, y).Close()
	}
	var points [][2]float32
	switch kind {
	case kindDiamond:
		points = [][2]float32{{0.5, 0}, {1, 0.5}, {0.5, 1}, {0, 0.5}}
	case kindTriangle:
		points = [][2]float32{{0.5, 0}, {1, 1}, {0, 1}}
	case kindHexagon:
		points = [][2]float32{{0.25, 0}, {0.75, 0}, {1, 0.5}, {0.75, 1}, {0.25, 1}, {0, 0.5}}
	case kindStar:
		for i := range 10 {
			radius := 0.5
			if i%2 != 0 {
				radius *= math.Sin(math.Pi/10) / math.Sin(3*math.Pi/10)
			}
			angle := -math.Pi/2 + float64(i)*math.Pi/5
			points = append(points, [2]float32{float32(0.5 + radius*math.Cos(angle)), float32(0.5 + radius*math.Sin(angle))})
		}
	}
	for i, point := range points {
		x, y := r.X+point[0]*r.W, r.Y+point[1]*r.H
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	return path.Close()
}

// paintShape 使用毫米线宽和圆角绘制空心或实心图形
func (a *item) paintShape(p *ui.Painter, r ui.Rect, unitsPerMM float32, ink ui.Color) {
	width := float32(a.Stroke) * unitsPerMM
	if a.Kind == kindRectangle {
		radius := float32(a.CornerRadius) * unitsPerMM
		if a.Filled {
			p.Fill(r, ink, radius)
		} else {
			p.Stroke(r, ink, radius, width)
		}
		return
	}
	if !a.Filled {
		r.X, r.Y, r.W, r.H = r.X+width/2, r.Y+width/2, r.W-width, r.H-width
	}
	path := shapePath(a.Kind, r)
	if a.Filled {
		p.FillPath(path, ink)
	} else {
		p.StrokePath(path, width, ink)
	}
}

// paintLine 在元素边界内绘制实线、虚线、点线、双实线或箭头
func (a *item) paintLine(p *ui.Painter, r ui.Rect, unitsPerMM float32, ink ui.Color) {
	width, y := float32(a.lineWidth())*unitsPerMM, r.Y+r.H/2
	switch a.LineStyle {
	case lineSolid:
		p.Line(r.X, y, r.X+r.W, y, width, ink)
	case lineDouble:
		p.Line(r.X, y-width, r.X+r.W, y-width, width, ink)
		p.Line(r.X, y+width, r.X+r.W, y+width, width, ink)
	case lineDashed, lineDotted:
		path := new(ui.Path).MoveTo(r.X+width/2, y).LineTo(r.X+r.W-width/2, y)
		if a.LineStyle == lineDashed {
			path.Cap(ui.ButtCap).Dash(0, 4*width, 2*width)
		} else {
			path.Cap(ui.RoundCap).Dash(0, 0.01*width, 3*width)
		}
		p.StrokePath(path, width, ink)
	case lineArrow, lineDoubleArrow:
		start, end := r.X, r.X+r.W
		head := min(r.W/2, max(r.H, width*3))
		p.FillPath(new(ui.Path).MoveTo(end, y).LineTo(end-head, r.Y).
			LineTo(end-head, r.Y+r.H).Close(), ink)
		end -= head / 2
		if a.LineStyle == lineDoubleArrow {
			p.FillPath(new(ui.Path).MoveTo(start, y).LineTo(start+head, r.Y).
				LineTo(start+head, r.Y+r.H).Close(), ink)
			start += head / 2
		}
		p.Line(start, y, end, y, width, ink)
	}
}

// shapeProperties 编辑图形线宽、填充和矩形圆角
func (e *editor) shapeProperties(c *ui.Context, a *item) {
	stroke, radius, filled := a.Stroke, a.CornerRadius, a.Filled
	ui.Row(c.Key("dimensions")).Gap(8).Children(func() {
		for i, field := range []struct {
			name  string
			value *float64
			low   float64
		}{
			{i18n.Tr("table.stroke"), &a.Stroke, minimumLineWidth},
			{i18n.Tr("shape.radius"), &a.CornerRadius, 0},
		} {
			if i == 1 && a.Kind != kindRectangle {
				continue
			}
			ui.Column(c.Key(field.name)).Grow(1).MinWidth(0).Gap(5).Children(func() {
				ui.Text(c, field.name).FontSize(10).TextColor(colors.Muted)
				inspectorNumber(c, field.name, "mm", 0, field.value, field.low, min(a.W, a.H)/2, 1, false)
			})
		}
	})
	ui.Row(c.Key("fill")).Height(34).AlignItems(ui.Center).Children(func() {
		ui.Text(c, i18n.Tr("shape.fill")).FontSize(12).Grow(1).TextColor(colors.Body)
		ui.Switch(c.Key("switch"), &a.Filled).Label(i18n.Tr("shape.fill"))
	})
	if stroke != a.Stroke || radius != a.CornerRadius || filled != a.Filled {
		e.changed()
	}
}

// lineProperties 编辑线型和毫米线宽并为箭头及双线保留所需高度
func (e *editor) lineProperties(c *ui.Context, a *item) {
	labels := []string{i18n.Tr("line.solid"), i18n.Tr("line.dashed"), i18n.Tr("line.dotted"),
		i18n.Tr("line.double"), i18n.Tr("line.arrow"), i18n.Tr("line.doubleArrow")}
	selected, width := labels[slices.Index(lineStyles[:], a.LineStyle)], a.lineWidth()
	ui.Column(c.Key("style")).Gap(5).Children(func() {
		ui.Text(c, i18n.Tr("line.style")).FontSize(10).TextColor(colors.Muted)
		if ui.Select(c.Key("picker"), &selected, labels).FillWidth().Height(34).Radius(7).
			FontSize(12).Label(i18n.Tr("line.style")).Changed() {
			a.Stroke, a.LineStyle = width, lineStyles[slices.Index(labels, selected)]
			if a.LineStyle == lineArrow || a.LineStyle == lineDoubleArrow {
				a.H = max(a.H, 3, 4*width)
			}
			a.constrain(e.label)
			e.changed()
		}
	})
	before := width
	ui.Column(c.Key("stroke")).Gap(5).Children(func() {
		ui.Text(c, i18n.Tr("table.stroke")).FontSize(10).TextColor(colors.Muted)
		high := min(a.W, e.label.Height)
		if a.LineStyle == lineDouble {
			high = min(high, e.label.Height/3)
		}
		inspectorNumber(c, i18n.Tr("table.stroke"), "mm", 0, &width, minimumLineWidth, high, 1, false)
	})
	if width != before {
		a.Stroke = width
		a.constrain(e.label)
		e.changed()
	}
}
