package editor

import (
	"bytes"
	"math"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTableDividerDrag 验证缩放下的行列拖动、光标、修改标记和取消操作
func TestTableDividerDrag(t *testing.T) {
	for _, test := range []struct {
		name   string
		x, y   float32
		dx, dy float64
		cursor ui.Cursor
		cancel bool
	}{
		{"列宽", 1.0 / 3, 0.25, 3, 0, ui.CursorResizeColumn, false},
		{"行高", 1.0 / 6, 0.5, 0, 2, ui.CursorResizeRow, false},
		{"交点", 1.0 / 3, 0.5, 3, 2, ui.CursorMove, false},
		{"取消", 1.0 / 3, 0.5, 3, 2, ui.CursorMove, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			l := newLabel()
			l.Items = []item{{Kind: kindTable, X: 4, Y: 4, W: 30, H: 16, Rows: 2, Columns: 3, Stroke: 0.3}}
			selection := &tableSelection{}
			e := &editor{label: l, selected: 0, zoom: 1.5, cells: selection,
				document: &documentFile{}, tabs: &documentTabs{}}
			tt := ui.NewTester(e.canvasView, 660, 480)
			tt.Frame()
			paper := e.paperRect(ui.Rect{W: 660, H: 480})
			box := l.Items[0].box(l, paper)
			x, y := box.X+test.x*box.W, box.Y+test.y*box.H
			dx, dy := float32(test.dx/l.Width)*paper.W, float32(test.dy/l.Height)*paper.H
			tt.Move(x, y)
			if tt.Cursor() != test.cursor {
				t.Fatalf("分隔线光标不正确：%v", tt.Cursor())
			}
			tt.Press(x, y)
			tt.Move(x+dx, y+dy)
			if e.drag == nil || e.drag.selectCells || e.document.edited {
				t.Fatal("拖动没有进入行列调整状态或提前标记修改")
			}
			if test.cancel {
				tt.CancelPress()
			} else {
				tt.Release(x+dx, y+dy)
			}
			a := &l.Items[0]
			wantX, wantY := 1.0/3, 0.5
			if !test.cancel {
				wantX += test.dx / a.W
				wantY += test.dy / a.H
			}
			if math.Abs(a.ColumnEdges.position(1, a.Columns)-wantX) > 1e-6 ||
				math.Abs(a.RowEdges.position(1, a.Rows)-wantY) > 1e-6 {
				t.Fatalf("行列尺寸不正确：行 %v，列 %v", a.RowEdges, a.ColumnEdges)
			}
			if a.X != 4 || a.Y != 4 || a.W != 30 || a.H != 16 || e.cells != selection ||
				e.drag != nil || e.document.edited == test.cancel {
				t.Fatal("拖动改变了表格外框、选区或错误处理了修改状态")
			}
		})
	}
}

// TestResizedTableTemplate 验证不等宽高表格的保存、快照、合并选区和打印布局
func TestResizedTableTemplate(t *testing.T) {
	l := newLabel()
	l.Items = []item{{Kind: kindTable, X: 2, Y: 4, W: 30, H: 16, Rows: 2, Columns: 3, Stroke: 0.3,
		RowEdges: tableAxis{0.25}, ColumnEdges: tableAxis{0.2, 0.7},
		Merges: []cellRange{{Top: 0, Left: 0, Bottom: 0, Right: 1}}}}
	snapshot := l.snapshot()
	l.Items[0].ColumnEdges = l.Items[0].ColumnEdges.moved(1, 3, 2, 30)
	var data bytes.Buffer
	if err := writeLabel(&data, snapshot); err != nil {
		t.Fatal(err)
	}
	reopened, err := readLabel(&data)
	if err != nil {
		t.Fatal(err)
	}
	a := &reopened.Items[0]
	if !slices.Equal(a.RowEdges, tableAxis{0.25}) || !slices.Equal(a.ColumnEdges, tableAxis{0.2, 0.7}) {
		t.Fatal("保存布局丢失或快照被后续拖动修改")
	}
	box := ui.Rect{W: 300, H: 160}
	if a.cellAt(box, 150, 60) != (cellRange{Top: 1, Left: 1, Bottom: 1, Right: 1}) ||
		a.rangeBox(box, a.Merges[0]) != (ui.Rect{W: 210, H: 40}) {
		t.Fatal("单元格命中或合并选区未使用调整后的尺寸")
	}
	if row, column := a.tableDividerAt(box, 60, 20); row != 0 || column != 0 {
		t.Fatal("合并单元格中隐藏的分隔线仍可拖动")
	}
	printed, err := reopened.render()
	if err != nil {
		t.Fatal(err)
	}
	scale := float64(l.DPI) / 25.4
	for _, point := range []struct {
		x, y float64
		ink  bool
	}{{8, 6, false}, {8, 12, true}, {23, 6, true}, {5, 8, true}, {5, 12, false}} {
		red, _, _, _ := printed.At(int(point.x*scale), int(point.y*scale)).RGBA()
		if (red < 0x8000) != point.ink {
			t.Errorf("打印布局在 (%g, %g) 毫米处不正确", point.x, point.y)
		}
	}
}

// TestTableDividerLimits 验证分隔线不能越界且异常布局会被模板校验拒绝
func TestTableDividerLimits(t *testing.T) {
	axis := tableAxis{0.2, 0.7}
	for _, delta := range []float64{-100, 100} {
		moved := axis.moved(1, 3, delta, 30)
		if !moved.valid(3) || moved[0] < 1.0/30 || moved[0] > 0.7-1.0/30 || axis[0] != 0.2 {
			t.Fatalf("拖动越界或修改了原布局：%v", moved)
		}
	}
	for _, edges := range []tableAxis{{0}, {1}, {0.6, 0.4}, {0.3, 0.5, 0.8}} {
		l := newLabel()
		l.Items = []item{{Kind: kindTable, X: 2, Y: 4, W: 30, H: 16, Rows: 2, Columns: 3, Stroke: 0.3,
			ColumnEdges: edges}}
		var data bytes.Buffer
		if err := writeLabel(&data, l); err != nil {
			t.Fatal(err)
		}
		if _, err := readLabel(&data); err == nil {
			t.Fatalf("模板校验接受了异常分隔线：%v", edges)
		}
	}
}
