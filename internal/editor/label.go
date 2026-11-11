package editor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
)

// itemKind 区分标签元素的内容类型
type itemKind string

const (
	// kindText 表示支持独立格式的文字
	kindText itemKind = "text"
	// kindImage 表示内嵌图片
	kindImage itemKind = "image"
	// kindBarcode 表示 Code 128 一维条码
	kindBarcode itemKind = "code128"
	// kindQR 表示二维码
	kindQR itemKind = "qr"
	// kindLine 表示支持独立线宽和多种线型的线条
	kindLine itemKind = "line"
	// kindTable 表示等宽等高单元格组成的表格
	kindTable itemKind = "table"
	// kindRectangle 表示可调圆角的矩形
	kindRectangle itemKind = "rectangle"
	// kindEllipse 表示椭圆或等宽高的圆形
	kindEllipse itemKind = "ellipse"
	// kindDiamond 表示菱形
	kindDiamond itemKind = "diamond"
	// kindTriangle 表示三角形
	kindTriangle itemKind = "triangle"
	// kindStar 表示五角星
	kindStar itemKind = "star"
	// kindHexagon 表示六边形
	kindHexagon itemKind = "hexagon"
	// minimumSide 是标签宽高的最小毫米数
	minimumSide = 10
	// maximumSide 是标签宽高的最大毫米数
	maximumSide = 200
	// minimumItem 是元素宽高的最小毫米数
	minimumItem = 1
	// minimumLineWidth 是横线和表格线宽的最小毫米数
	minimumLineWidth = 0.2
)

// label 保存标签尺寸、打印分辨率和按绘制顺序排列的元素
type label struct {
	Width  float64 `json:"widthMm"`
	Height float64 `json:"heightMm"`
	DPI    int     `json:"dpi"`
	Items  []item  `json:"items"`
	// title 是未保存标签在编辑器中显示的名称
	title string
}

// item 保存元素内容、毫米坐标、图形样式和内存中的绘制资源
type item struct {
	Kind itemKind `json:"kind"`
	X    float64  `json:"xMm"`
	Y    float64  `json:"yMm"`
	W    float64  `json:"widthMm"`
	H    float64  `json:"heightMm"`
	labelText
	Data    []byte         `json:"image,omitempty"`
	Rows    int            `json:"rows,omitempty"`
	Columns int            `json:"columns,omitempty"`
	Stroke  float64        `json:"strokeMm,omitempty"`
	Cells   [][]*tableCell `json:"cells,omitempty"`
	Merges  []cellRange    `json:"merges,omitempty"`

	CornerRadius float64   `json:"cornerRadiusMm,omitempty"`
	Filled       bool      `json:"filled,omitempty"`
	LineStyle    lineStyle `json:"lineStyle,omitempty"`

	bitmap *ui.Bitmap
	symbol *symbol
	issue  error
}

// symbol 保存条码的模块数和每行连续黑块的位置
type symbol struct {
	cols, rows int
	bars       []ui.Rect
}

// newLabel 创建 50 × 30 毫米空白标签
func newLabel() *label {
	return &label{Width: 50, Height: 30, DPI: 203}
}

// snapshot 复制标签和可编辑的元素列表
func (l *label) snapshot() *label {
	copy := *l
	copy.Items = slices.Clone(l.Items)
	return &copy
}

// readLabel 解析并校验标签模板
func readLabel(r io.Reader) (*label, error) {
	var l label
	dec := json.NewDecoder(r)
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf(i18n.Tr("error.templateFormat"), err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New(i18n.Tr("error.templateExtra"))
	}
	if !within(l.Width, minimumSide, maximumSide) || !within(l.Height, minimumSide, maximumSide) || (l.DPI != 203 && l.DPI != 300 && l.DPI != 600) {
		return nil, errors.New(i18n.Tr("error.templatePaper"))
	}
	for i := range l.Items {
		e := &l.Items[i]
		if !within(e.W, minimumItem, l.Width) || !within(e.H, e.minimumHeight(), l.Height) ||
			!within(e.X, 0, l.Width-e.W+1e-8) || !within(e.Y, 0, l.Height-e.H+1e-8) {
			return nil, fmt.Errorf(i18n.Tr("error.itemOutside"), i+1)
		}
		switch e.Kind {
		case kindText:
			if !within(e.FontSize, 1, 40) {
				return nil, fmt.Errorf(i18n.Tr("error.textSize"), i+1)
			}
			if e.BlockAlign > ui.End {
				return nil, fmt.Errorf(i18n.Tr("error.textAlign"), i+1)
			}
		case kindImage:
			bitmap, err := decodeImage(e.Data)
			if err != nil {
				return nil, fmt.Errorf(i18n.Tr("error.itemDetail"), i+1, err)
			}
			e.bitmap = bitmap
		case kindTable:
			if !e.validTable() {
				return nil, fmt.Errorf(i18n.Tr("error.table"), i+1)
			}
		case kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon:
			if !e.validShape() {
				return nil, fmt.Errorf(i18n.Tr("error.shape"), i+1)
			}
		case kindLine:
			if !e.validLine() {
				return nil, fmt.Errorf(i18n.Tr("error.line"), i+1)
			}
		case kindBarcode, kindQR:
		default:
			return nil, fmt.Errorf(i18n.Tr("error.itemKind"), i+1)
		}
	}
	return &l, nil
}

// writeLabel 将模板保存为带有毫米字段的 JSON
func writeLabel(w io.Writer, l *label) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(l)
}

// within 检查数值是否在指定闭区间内
func within(value, low, high float64) bool {
	return value >= low && value <= high
}

// decodeImage 读取模板内嵌的 PNG 或 JPEG 图像
func decodeImage(data []byte) (*ui.Bitmap, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New(i18n.Tr("error.imageType"))
	}
	if int64(config.Width)*int64(config.Height) > 24_000_000 {
		return nil, errors.New(i18n.Tr("error.imagePixels"))
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf(i18n.Tr("error.imageDecode"), err)
	}
	return ui.NewBitmap(decoded), nil
}

// minimumHeight 返回元素高度允许的最小毫米数
func (e *item) minimumHeight() float64 {
	if e.Kind == kindLine {
		factor := 1.0
		if e.LineStyle == lineDouble {
			factor = 3
		}
		return max(minimumLineWidth, factor*e.Stroke)
	}
	return minimumItem
}

// constrain 将元素尺寸和位置限制在当前标签范围内
func (e *item) constrain(l *label) {
	e.W = min(max(e.W, minimumItem), l.Width)
	e.H = min(max(e.H, e.minimumHeight()), l.Height)
	switch e.Kind {
	case kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon:
		e.Stroke = min(e.Stroke, min(e.W, e.H)/2)
		e.CornerRadius = min(e.CornerRadius, min(e.W, e.H)/2)
	case kindLine:
		factor := 1.0
		if e.LineStyle == lineDouble {
			factor = 3
		}
		e.Stroke = min(e.Stroke, e.W, e.H/factor)
	}
	e.X = min(max(e.X, 0), l.Width-e.W)
	e.Y = min(max(e.Y, 0), l.Height-e.H)
}

// prepareSymbol 校验码值并缓存用于绘制的黑色模块
func (e *item) prepareSymbol(g *platform.Graphics) {
	e.symbol, e.issue = nil, nil
	if e.Value == "" {
		e.issue = errors.New(i18n.Tr("error.encodeEmpty"))
		return
	}
	if e.Kind == kindBarcode {
		for _, c := range e.Value {
			if c < 32 || c > 126 {
				e.issue = errors.New(i18n.Tr("error.barcodeCharset"))
				return
			}
		}
		if len(e.Value) > 80 {
			e.issue = errors.New(i18n.Tr("error.barcodeLength"))
			return
		}
	}
	if e.Kind == kindQR && len(e.Value) > 512 {
		e.issue = errors.New(i18n.Tr("error.qrLength"))
		return
	}
	im, err := g.SymbolImage(e.Kind == kindBarcode, e.Value)
	if err != nil {
		e.issue = err
		return
	}
	bounds := im.Bounds()
	s := &symbol{cols: bounds.Dx(), rows: bounds.Dy()}
	padding, firstRow, lastRow := 0, 0, bounds.Dy()
	if e.Kind == kindQR {
		padding = 4
		s.cols += 2 * padding
		s.rows += 2 * padding
	} else {
		s.rows = 1
		firstRow = bounds.Dy() / 2
		lastRow = firstRow + 1
	}
	for y := firstRow; y < lastRow; y++ {
		for x := 0; x < bounds.Dx(); {
			if !darkPixel(im, bounds.Min.X+x, bounds.Min.Y+y) {
				x++
				continue
			}
			left := x
			for x < bounds.Dx() && darkPixel(im, bounds.Min.X+x, bounds.Min.Y+y) {
				x++
			}
			s.bars = append(s.bars, ui.Rect{X: float32(left + padding), Y: float32(y - firstRow + padding), W: float32(x - left), H: 1})
		}
	}
	e.symbol = s
}

// darkPixel 判断条码图像中的模块是否为黑色
func darkPixel(im image.Image, x, y int) bool {
	r, g, b, a := im.At(x, y).RGBA()
	return a > 32767 && r+g+b < 3*32768
}

// box 将元素的毫米坐标映射到纸张绘制区域
func (e *item) box(l *label, paper ui.Rect) ui.Rect {
	sx, sy := paper.W/float32(l.Width), paper.H/float32(l.Height)
	return ui.Rect{X: paper.X + float32(e.X)*sx, Y: paper.Y + float32(e.Y)*sy, W: float32(e.W) * sx, H: float32(e.H) * sy}
}

// moduleWidth 计算条码可以容纳的模块宽度
func (e *item) moduleWidth(rect ui.Rect) float32 {
	wide := rect.W / float32(e.symbol.cols)
	if e.Kind == kindQR {
		wide = min(wide, rect.H/float32(e.symbol.rows))
	}
	return wide
}

// paint 在画布或打印位图上绘制标签内容
func (l *label) paint(p *ui.Painter, paper ui.Rect, printing bool) {
	black, white := ui.Hex("#000000"), ui.Hex("#FFFFFF")
	p.Fill(paper, white, 0)
	p.Clip(paper, 0, func() {
		unitsPerMM := paper.H / float32(l.Height)
		for i := range l.Items {
			e := &l.Items[i]
			rect := e.box(l, paper)
			switch e.Kind {
			case kindText:
				e.paintText(p, rect, unitsPerMM, black)
			case kindImage:
				p.Image(e.bitmap, rect, ui.Contain)
			case kindLine:
				e.paintLine(p, rect, unitsPerMM, black)
			case kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon:
				e.paintShape(p, rect, unitsPerMM, black)
			case kindTable:
				e.paintTable(p, rect, unitsPerMM, black)
			case kindBarcode, kindQR:
				if e.symbol != nil {
					e.paintSymbol(p, rect, black, white, printing)
				}
			}
		}
	})
}

// paintSymbol 绘制白色静区和黑色模块并在打印时对齐点阵
func (e *item) paintSymbol(p *ui.Painter, rect ui.Rect, black, white ui.Color, printing bool) {
	step := e.moduleWidth(rect)
	if printing {
		step = float32(math.Floor(float64(step)))
	}
	s := e.symbol
	w, h := step*float32(s.cols), step*float32(s.rows)
	if e.Kind == kindBarcode {
		h = rect.H
	}
	x, y := rect.X+(rect.W-w)/2, rect.Y+(rect.H-h)/2
	rowHeight := h / float32(s.rows)
	if printing {
		x = float32(math.Round(float64(x)))
		y = float32(math.Round(float64(y)))
		if e.Kind == kindBarcode {
			rowHeight = float32(math.Floor(float64(rowHeight)))
		}
	}
	p.Fill(rect, white, 0)
	for _, bar := range s.bars {
		p.Fill(ui.Rect{X: x + step*bar.X, Y: y + rowHeight*bar.Y, W: step * bar.W, H: rowHeight}, black, 0)
	}
}

// render 按选定 DPI 生成打印位图并检查码的点阵大小
func (l *label) render() (*image.RGBA, error) {
	if len(l.Items) == 0 {
		return nil, errors.New(i18n.Tr("error.emptyLabel"))
	}
	width := int(math.Round(l.Width * float64(l.DPI) / 25.4))
	height := int(math.Round(l.Height * float64(l.DPI) / 25.4))
	page := ui.Rect{W: float32(width), H: float32(height)}
	for i := range l.Items {
		e := &l.Items[i]
		if e.issue != nil {
			return nil, fmt.Errorf(i18n.Tr("error.itemDetail"), i+1, e.issue)
		}
		if e.symbol != nil && e.moduleWidth(e.box(l, page)) < 2 {
			return nil, fmt.Errorf(i18n.Tr("error.modulesNarrow"), i+1)
		}
	}
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { l.paint(p, r, true) })
	}, width, height, 1)
	return img, nil
}

// printPage 将标签渲染为带实际纸张尺寸的页面
func (l *label) printPage() (platform.Page, error) {
	raster, err := l.render()
	if err != nil {
		return platform.Page{}, err
	}
	return platform.Page{Raster: raster, WidthMM: l.Width, HeightMM: l.Height}, nil
}
