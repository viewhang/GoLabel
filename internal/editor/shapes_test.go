package editor

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestShapeRendering 验证图形的保存重开、空心填充、圆角和打印边界
func TestShapeRendering(t *testing.T) {
	for _, kind := range []itemKind{kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon} {
		for _, filled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/填充%t", kind, filled), func(t *testing.T) {
				l := newLabel()
				a := item{Kind: kind, X: 2, Y: 2, W: 20, H: 16, Stroke: 0.4, Filled: filled}
				if kind == kindRectangle {
					a.CornerRadius = 4
				}
				l.Items = []item{a}
				var saved bytes.Buffer
				if err := writeLabel(&saved, l); err != nil {
					t.Fatal(err)
				}
				loaded, err := readLabel(&saved)
				if err != nil {
					t.Fatal(err)
				}
				im, err := loaded.render()
				if err != nil {
					t.Fatal(err)
				}
				original, err := l.render()
				if err != nil || !bytes.Equal(original.Pix, im.Pix) {
					t.Fatalf("重开后图形绘制发生变化：%v", err)
				}
				scale := float64(l.DPI) / 25.4
				dark := func(x, y float64) bool {
					red, _, _, _ := im.At(int(x*scale), int(y*scale)).RGBA()
					return red < 0x8000
				}
				if dark(12, 10) != filled {
					t.Fatal("图形填充状态没有反映在打印图像中")
				}
				if kind == kindRectangle && dark(2.1, 2.1) {
					t.Fatal("圆角矩形仍然绘制了直角")
				}
				ink := 0
				for y := range im.Bounds().Dy() {
					for x := range im.Bounds().Dx() {
						red, _, _, _ := im.At(x, y).RGBA()
						if red >= 0x8000 {
							continue
						}
						ink++
						if float64(x) < a.X*scale-1 || float64(x) > (a.X+a.W)*scale+1 ||
							float64(y) < a.Y*scale-1 || float64(y) > (a.Y+a.H)*scale+1 {
							t.Fatal("图形绘制超出了元素边界")
						}
					}
				}
				if ink < 30 {
					t.Fatal("打印图像缺少图形")
				}
				loaded.Items[0].W, loaded.Items[0].H = 3, 2
				loaded.Items[0].constrain(loaded)
				if !loaded.Items[0].validShape() {
					t.Fatal("缩小图形后圆角或线宽超出允许范围")
				}
			})
		}
	}
}

// TestLineStyles 验证线型的保存和打印图案并兼容旧横线
func TestLineStyles(t *testing.T) {
	for _, style := range lineStyles {
		t.Run(string(style), func(t *testing.T) {
			l := newLabel()
			l.Items = []item{{Kind: kindLine, X: 2, Y: 2, W: 30, H: 4, Stroke: 0.4, LineStyle: style}}
			var saved bytes.Buffer
			if err := writeLabel(&saved, l); err != nil {
				t.Fatal(err)
			}
			loaded, err := readLabel(&saved)
			if err != nil || loaded.Items[0].LineStyle != style {
				t.Fatalf("保存后没有保留线型：%v", err)
			}
			im, err := loaded.render()
			if err != nil {
				t.Fatal(err)
			}
			scale := float64(l.DPI) / 25.4
			runs, previous := 0, false
			for x := int(2 * scale); x < int(32*scale); x++ {
				red, _, _, _ := im.At(x, int(4*scale)).RGBA()
				dark := red < 0x8000
				if dark && !previous {
					runs++
				}
				previous = dark
			}
			if (style == lineDashed || style == lineDotted) && runs < 4 {
				t.Fatal("虚线或点线没有形成间隔")
			}
			if style == lineDouble {
				red, _, _, _ := im.At(int(10*scale), int(4*scale)).RGBA()
				top, _, _, _ := im.At(int(10*scale), int(3.6*scale)).RGBA()
				bottom, _, _, _ := im.At(int(10*scale), int(4.4*scale)).RGBA()
				if red < 0x8000 || top >= 0x8000 || bottom >= 0x8000 {
					t.Fatal("双实线中间没有留白或缺少其中一条线")
				}
			}
			if style == lineArrow || style == lineDoubleArrow {
				red, _, _, _ := im.At(int(29*scale), int(3*scale)).RGBA()
				if red >= 0x8000 {
					t.Fatal("箭头没有绘制右侧箭头头部")
				}
				if style == lineDoubleArrow {
					red, _, _, _ := im.At(int(5*scale), int(3*scale)).RGBA()
					if red >= 0x8000 {
						t.Fatal("双向箭头缺少左侧箭头头部")
					}
				}
			}
		})
	}
	l := newLabel()
	l.Items = []item{{Kind: kindLine, X: 2, Y: 2, W: 25, H: 0.3}}
	original, err := l.render()
	if err != nil {
		t.Fatal(err)
	}
	l.Items[0].Stroke = 0.3
	updated, err := l.render()
	if err != nil || !bytes.Equal(original.Pix, updated.Pix) {
		t.Fatalf("独立线宽改变了旧横线的打印位置或粗细：%v", err)
	}
}

// TestGraphicProperties 验证工具添加、圆角小数输入、填充和线型选择
func TestGraphicProperties(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 900)
	for _, kind := range []itemKind{kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon} {
		if err := tt.Click(kindTitle(kind)); err != nil {
			t.Fatal(err)
		}
		if a := e.selectedItem(); a == nil || a.Kind != kind {
			t.Fatal("点击图形工具没有添加并选中相应图形")
		}
	}
	e.selectItem(0)
	tt.Frame()
	if err := tt.Click("圆角半径（mm）"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("2.5")
	if e.label.Items[0].CornerRadius != 2.5 {
		t.Fatal("圆角半径没有保留小数")
	}
	switchControl, ok := tt.AccessByRole(ui.RoleSwitch, "填充")
	if !ok {
		t.Fatal("找不到图形填充开关")
	}
	tt.ClickAt(switchControl.Bounds.X+switchControl.Bounds.W/2, switchControl.Bounds.Y+switchControl.Bounds.H/2)
	if !e.label.Items[0].Filled {
		t.Fatal("填充开关没有写入图形属性")
	}
	saveImage(t, "golabel-shape-properties", tt)
	if err := tt.Click("线条"); err != nil {
		t.Fatal(err)
	}
	labels := []string{"实线", "虚线", "点线", "双实线", "单向箭头", "双向箭头"}
	for _, title := range []string{"虚线", "双实线", "单向箭头", "双向箭头", "点线", "实线"} {
		current := labels[slices.Index(lineStyles[:], e.selectedItem().LineStyle)]
		if err := tt.Click(current); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(title); err != nil {
			t.Fatal(err)
		}
		if e.selectedItem().LineStyle != lineStyles[slices.Index(labels, title)] {
			t.Fatal("选择线型后没有更新线条属性")
		}
	}
	saveImage(t, "golabel-line-properties", tt)
}

// TestThinStrokeWidth 验证线条、图形和表格可输入并保存零点一毫米线宽
func TestThinStrokeWidth(t *testing.T) {
	for _, kind := range []itemKind{kindLine, kindTable, kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon} {
		t.Run(string(kind), func(t *testing.T) {
			l := newLabel()
			l.Items = []item{{Kind: kind, X: 2, Y: 2, W: 20, H: 10, Rows: 2, Columns: 2, Stroke: 0.3}}
			e := &editor{label: l, selected: 0, zoom: 1}
			tt := ui.NewTester(e.view, 1280, 900)
			if err := tt.Click("线宽（mm）"); err != nil {
				t.Fatal(err)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type("0.1")
			if l.Items[0].Stroke != 0.1 {
				t.Fatalf("零点一毫米线宽未生效：%g", l.Items[0].Stroke)
			}
			var saved bytes.Buffer
			if err := writeLabel(&saved, l); err != nil {
				t.Fatal(err)
			}
			reopened, err := readLabel(&saved)
			if err != nil || reopened.Items[0].Stroke != 0.1 {
				t.Fatalf("零点一毫米线宽无法保存重开：%v", err)
			}
		})
	}
}

// TestInvalidGraphicData 验证文件输入拒绝无效线型、线宽和圆角
func TestInvalidGraphicData(t *testing.T) {
	for _, a := range []item{
		{Kind: kindRectangle, W: 10, H: 10, Stroke: 0.3, CornerRadius: 6},
		{Kind: kindStar, W: 10, H: 10, Stroke: -1},
		{Kind: kindLine, W: 10, H: 2, Stroke: 0.3, LineStyle: "unknown"},
		{Kind: kindLine, W: 10, H: 2, Stroke: 1, LineStyle: lineDouble},
	} {
		l := newLabel()
		l.Items = []item{a}
		var saved bytes.Buffer
		if err := writeLabel(&saved, l); err != nil {
			t.Fatal(err)
		}
		if _, err := readLabel(&saved); err == nil {
			t.Fatalf("读取时接受了无效图形数据：%+v", a)
		}
	}
}

// TestGraphicsPrintSheet 渲染包含全部图形和线型的打印检查图
func TestGraphicsPrintSheet(t *testing.T) {
	l := &label{Width: 100, Height: 70, DPI: 300}
	for i, kind := range []itemKind{kindRectangle, kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon} {
		a := item{Kind: kind, X: 3 + float64(i%3)*33, Y: 3 + float64(i/3)*20, W: 26, H: 15, Stroke: 0.4}
		if kind == kindRectangle {
			a.CornerRadius = 3
		}
		l.Items = append(l.Items, a)
	}
	for i, style := range lineStyles {
		l.Items = append(l.Items, item{Kind: kindLine, X: 3 + float64(i%2)*50, Y: 44 + float64(i/2)*8,
			W: 42, H: 4, Stroke: 0.4, LineStyle: style})
	}
	im, err := l.render()
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds() != image.Rect(0, 0, int(math.Round(100*300/25.4)), int(math.Round(70*300/25.4))) {
		t.Fatal("打印图像没有使用标签毫米尺寸和 DPI")
	}
	bitmap := ui.NewBitmap(im)
	tt := ui.NewTester(func(c *ui.Context) { ui.Image(c, bitmap).Fill() }, im.Bounds().Dx(), im.Bounds().Dy())
	saveImage(t, "golabel-graphics-print-sheet", tt)
}

// TestGraphicPaletteScrollAndDrag 验证小窗口可滚动选择图形并将图形拖入画布
func TestGraphicPaletteScrollAndDrag(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 980, 660)
	tt.Scroll(100, 200, 0, 600)
	if err := tt.Click("五角星"); err != nil {
		t.Fatal(err)
	}
	if len(e.label.Items) != 1 || e.label.Items[0].Kind != kindStar {
		t.Fatal("小窗口滚动后无法添加图形")
	}
	tool, ok := tt.AccessByRole(ui.RoleButton, "菱形")
	if !ok {
		t.Fatal("找不到菱形工具")
	}
	canvas, ok := tt.Find("标签画布")
	if !ok {
		t.Fatal("找不到画布")
	}
	paper := e.paperRect(ui.Rect{W: canvas.W, H: canvas.H})
	x, y := tool.Bounds.X+tool.Bounds.W/2, tool.Bounds.Y+tool.Bounds.H/2
	dropX, dropY := canvas.X+paper.X+paper.W*7/50, canvas.Y+paper.Y+paper.H*8/30
	tt.Press(x, y)
	tt.Move(x+12, y)
	tt.Move(dropX, dropY)
	tt.Release(dropX, dropY)
	if len(e.label.Items) != 2 || e.label.Items[1].Kind != kindDiamond ||
		math.Abs(e.label.Items[1].X-7) > 0.1 || math.Abs(e.label.Items[1].Y-8) > 0.1 {
		t.Fatal("拖入图形没有使用画布落点")
	}
	saveImage(t, "golabel-shapes-small-window", tt)
}
