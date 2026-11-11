package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestCanvasDrawsHeightRulerOnFirstFrame 验证默认窗口的左侧标尺首帧就显示高度
func TestCanvasDrawsHeightRulerOnFirstFrame(t *testing.T) {
	// width 和 height 是默认窗口扣除顶栏和两侧面板后的画布尺寸
	const width, height = 700, 738
	ink := colors.RulerText
	colors.RulerText = ui.RGB(255, 0, 0)
	t.Cleanup(func() { colors.RulerText = ink })

	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	img := ui.NewTester(e.canvasView, width, height).Image()
	paper := e.paperRect(ui.Rect{W: width, H: height})
	centerY := int(paper.Y + paper.H/2)
	for y := centerY - 24; y < centerY+24; y++ {
		for x := int(paper.X) - 42; x < int(paper.X)-9; x++ {
			pixel := img.RGBAAt(x, y)
			if int(pixel.R)-int(pixel.G) > 32 {
				return
			}
		}
	}
	t.Fatal("首次绘制没有显示左侧标尺数字")
}

// TestCanvasDragGuides 验证缩放下的对齐吸附及取消拖动后的状态
func TestCanvasDragGuides(t *testing.T) {
	// width 和 height 是测试画布的宽高
	const width, height = 660, 480
	l := newLabel()
	l.Items = []item{
		{Kind: kindText, X: 4, Y: 4, W: 10, H: 6},
		{Kind: kindText, X: 25, Y: 10, W: 10, H: 6},
	}
	e := &editor{label: l, zoom: 1.5, selected: -1}
	tt := ui.NewTester(e.canvasView, width, height)
	paper := e.paperRect(ui.Rect{W: width, H: height})
	box := l.Items[0].box(l, paper)
	x, y := box.X+box.W/2, box.Y+box.H/2
	tt.Press(x, y)
	tt.Move(x+float32(11/l.Width)*paper.W-3, y+float32(6/l.Height)*paper.H+3)
	if a := l.Items[0]; a.X != 15 || a.Y != 10 ||
		e.drag.guideX != (alignmentGuide{at: 25, visible: true}) || e.drag.guideY != (alignmentGuide{at: 10, visible: true}) {
		t.Fatalf("边缘对齐未生效：位置 (%g, %g)，对齐线 %+v、%+v", a.X, a.Y, e.drag.guideX, e.drag.guideY)
	}

	tt.Move(x+float32(11/l.Width)*paper.W-9, y+float32(6/l.Height)*paper.H+9)
	if e.drag.guideX.visible || e.drag.guideY.visible {
		t.Fatal("移出吸附范围后仍显示对齐线")
	}
	tt.CancelPress()
	if a := l.Items[0]; e.drag != nil || a.X != 4 || a.Y != 4 {
		t.Fatal("取消拖动后未还原元素位置或清除对齐线")
	}
}
