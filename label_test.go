package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestLineAndTableTemplate 验证横线和表格的模板往返及打印渲染
func TestLineAndTableTemplate(t *testing.T) {
	l := newLabel()
	l.Items = []item{
		{Kind: kindLine, X: 2, Y: 2, W: 20, H: minimumLineWidth},
		{Kind: kindTable, X: 2, Y: 5, W: 20, H: 10, Rows: 2, Columns: 2, Stroke: 0.3},
	}
	cell := l.Items[1].cellValue(0, 1)
	cell.Value = "品名"
	l.Items[1].setCell(0, 1, cell)
	snapshot := l.snapshot()
	cell.Value = "数量"
	l.Items[1].setCell(0, 1, cell)

	var data bytes.Buffer
	if err := writeLabel(&data, snapshot); err != nil {
		t.Fatal(err)
	}
	reopened, err := readLabel(&data)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Items[0].H != minimumLineWidth || reopened.Items[1].cellValue(0, 1).Value != "品名" {
		t.Fatal("重新打开模板后横线或表格内容有误")
	}
	printed, err := reopened.render()
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct {
		x, y int
		ink  bool
	}{{80, 16, true}, {95, 55, true}, {55, 79, true}, {55, 55, false}} {
		red, _, _, _ := printed.At(point.x, point.y).RGBA()
		if (red < 0x8000) != point.ink {
			t.Errorf("打印位图在 (%d, %d) 的像素有误", point.x, point.y)
		}
	}
}

// TestMergedTableTemplate 验证合并区域和原单元格内容可保存且打印线条正确
func TestMergedTableTemplate(t *testing.T) {
	l := newLabel()
	l.Items = []item{{Kind: kindTable, X: 2, Y: 5, W: 20, H: 10, Rows: 2, Columns: 2, Stroke: 0.3}}
	table := &l.Items[0]
	cell := table.cellValue(0, 1)
	cell.Value = "保留"
	table.setCell(0, 1, cell)
	top := cellRange{Top: 0, Left: 0, Bottom: 0, Right: 1}
	table.mergeCells(top)
	snapshot := l.snapshot()
	table.mergeCells(cellRange{Top: 0, Left: 0, Bottom: 1, Right: 1})

	var data bytes.Buffer
	if err := writeLabel(&data, snapshot); err != nil {
		t.Fatal(err)
	}
	reopened, err := readLabel(&data)
	if err != nil {
		t.Fatal(err)
	}
	saved := &reopened.Items[0]
	if len(saved.Merges) != 1 || saved.Merges[0] != top || saved.cellValue(0, 1).Value != "保留" {
		t.Fatal("重开模板后合并区域或原单元格文字丢失")
	}
	printed, err := reopened.render()
	if err != nil {
		t.Fatal(err)
	}
	topRed, _, _, _ := printed.At(95, 55).RGBA()
	bottomRed, _, _, _ := printed.At(95, 95).RGBA()
	if topRed < 0x8000 || bottomRed >= 0x8000 {
		t.Fatal("打印位图未正确去除合并区内部的分隔线")
	}
}

// TestTableCellTextFormat 验证旧单元格可读取且新格式可保存和打印
func TestTableCellTextFormat(t *testing.T) {
	source := strings.NewReader(`{"widthMm":50,"heightMm":30,"dpi":203,"items":[
		{"kind":"table","xMm":2,"yMm":5,"widthMm":30,"heightMm":12,"rows":1,"columns":2,"strokeMm":0.3,"cells":[["X"]]}
	]}`)
	l, err := readLabel(source)
	if err != nil {
		t.Fatal(err)
	}
	table := &l.Items[0]
	cell := table.cellValue(0, 0)
	if cell.Value != "X" || cell.FontSize != tableFontSize || cell.BlockAlign != ui.Center {
		t.Fatal("旧版单元格未保留原文字和默认排版")
	}
	edit := newTextEdit(cell)
	formatText(edit.doc, edit.doc.ToggleBold)
	formatText(edit.doc, func() { edit.doc.SetAlign(ui.End) })
	edit.writeTo(&cell)
	cell.BlockAlign = ui.End
	table.setCell(0, 0, cell)

	var data bytes.Buffer
	if err := writeLabel(&data, l); err != nil {
		t.Fatal(err)
	}
	reopened, err := readLabel(&data)
	if err != nil {
		t.Fatal(err)
	}
	saved := reopened.Items[0].cellValue(0, 0)
	format := newTextEdit(saved).doc.Formatting()
	if saved.BlockAlign != ui.End || format.Paragraph.Align != ui.End || format.Attributes.Weight != 700 {
		t.Fatal("重新打开后未保留单元格对齐或加粗")
	}
	printed, err := reopened.render()
	if err != nil {
		t.Fatal(err)
	}
	hasInk := false
	for y := 44; y < 132; y++ {
		for x := 20; x < 132; x++ {
			red, _, _, _ := printed.At(x, y).RGBA()
			if red < 0x8000 {
				hasInk = true
				if x < 76 || y < 88 {
					t.Fatal("单元格文字未按右下对齐打印")
				}
			}
		}
	}
	if !hasInk {
		t.Fatal("打印位图缺少单元格文字")
	}
}
