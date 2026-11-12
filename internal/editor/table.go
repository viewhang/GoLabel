package editor

import (
	"encoding/json"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

// tableCell 保存与普通文字共用格式的单元格内容
type tableCell labelText

// UnmarshalJSON 读取单元格富文本并兼容旧版字符串
func (c *tableCell) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		text := labelText{FontSize: tableFontSize, BlockAlign: ui.Center}
		if err := json.Unmarshal(data, &text.Value); err != nil {
			return err
		}
		*c = tableCell(text)
		return nil
	}
	return json.Unmarshal(data, (*labelText)(c))
}

const (
	// maximumTableCount 是表格的最大行数和列数
	maximumTableCount = 20
	// maximumTableStroke 是表格线宽的最大毫米数
	maximumTableStroke = 2
	// tableFontSize 是表格文字字号的毫米数
	tableFontSize = 2.5
	// tableTextInset 是单元格文字与左右边缘的毫米距离
	tableTextInset = 0.5
)

// cellRange 表示包含首尾行列的矩形单元格区域
type cellRange struct {
	Top    int `json:"top"`
	Left   int `json:"left"`
	Bottom int `json:"bottom"`
	Right  int `json:"right"`
}

// tableSelection 保存拖选或 Shift 选择的起点和终点
type tableSelection struct {
	start, end cellRange
}

// tableAxis 保存行或列的内部分隔线比例，空值表示等分
type tableAxis []float64

// position 返回指定分隔线或外边缘相对表格边长的位置
func (axis tableAxis) position(index, count int) float64 {
	if index == 0 || index == count || len(axis) == 0 {
		return float64(index) / float64(count)
	}
	return axis[index-1]
}

// valid 检查分隔线数量及其在表格内部的递增顺序
func (axis tableAxis) valid(count int) bool {
	if len(axis) == 0 {
		return true
	}
	if len(axis) != count-1 {
		return false
	}
	previous := 0.0
	for _, edge := range axis {
		if !(edge > previous && edge < 1) {
			return false
		}
		previous = edge
	}
	return true
}

// cellAt 返回相对位置所在的行或列序号
func (axis tableAxis) cellAt(position float64, count int) int {
	index := 0
	for index < count-1 && position >= axis.position(index+1, count) {
		index++
	}
	return index
}

// moved 按毫米位移调整分隔线并保留相邻单元格及已有快照
func (axis tableAxis) moved(index, count int, delta, size float64) tableAxis {
	start := axis.position(index, count)
	left, right := axis.position(index-1, count), axis.position(index+1, count)
	gap := min(minimumItem/size, start-left, right-start)
	next := min(right-gap, max(left+gap, start+roundMM(delta)/size))
	if next == start {
		return axis
	}
	edges := make(tableAxis, count-1)
	for i := range edges {
		edges[i] = axis.position(i+1, count)
	}
	edges[index-1] = next
	return edges
}

// overlaps 判断两个单元格区域是否有交集
func (r cellRange) overlaps(other cellRange) bool {
	return r.Top <= other.Bottom && other.Top <= r.Bottom && r.Left <= other.Right && other.Left <= r.Right
}

// union 返回包含两个区域的最小矩形
func (r cellRange) union(other cellRange) cellRange {
	return cellRange{Top: min(r.Top, other.Top), Left: min(r.Left, other.Left),
		Bottom: max(r.Bottom, other.Bottom), Right: max(r.Right, other.Right)}
}

// multiple 判断区域是否包含多个基础单元格
func (r cellRange) multiple() bool {
	return r.Top != r.Bottom || r.Left != r.Right
}

// bounds 扩展选区以完整覆盖相交的合并单元格
func (s tableSelection) bounds(a *item) cellRange {
	area := s.start.union(s.end)
	for {
		previous := area
		for _, merged := range a.Merges {
			if area.overlaps(merged) {
				area = area.union(merged)
			}
		}
		if area == previous {
			return area
		}
	}
}

// validTable 检查表格的行列数、分隔线、线宽、单元格和合并区域
func (e *item) validTable() bool {
	if e.Rows < 1 || e.Rows > maximumTableCount || e.Columns < 1 || e.Columns > maximumTableCount ||
		!within(e.Stroke, minimumLineWidth, maximumTableStroke) || len(e.Cells) > maximumTableCount ||
		!e.RowEdges.valid(e.Rows) || !e.ColumnEdges.valid(e.Columns) {
		return false
	}
	for _, row := range e.Cells {
		if len(row) > maximumTableCount {
			return false
		}
		for _, cell := range row {
			if cell != nil && (!within(cell.FontSize, 1, 40) || cell.BlockAlign > ui.End) {
				return false
			}
		}
	}
	for i, area := range e.Merges {
		if area.Top < 0 || area.Left < 0 || area.Bottom < area.Top || area.Right < area.Left ||
			area.Bottom >= e.Rows || area.Right >= e.Columns || !area.multiple() ||
			slices.ContainsFunc(e.Merges[:i], area.overlaps) {
			return false
		}
	}
	return true
}

// rangeBox 计算单元格区域在画布或打印页中的边界
func (e *item) rangeBox(table ui.Rect, area cellRange) ui.Rect {
	left := table.X + float32(e.ColumnEdges.position(area.Left, e.Columns))*table.W
	right := table.X + float32(e.ColumnEdges.position(area.Right+1, e.Columns))*table.W
	top := table.Y + float32(e.RowEdges.position(area.Top, e.Rows))*table.H
	bottom := table.Y + float32(e.RowEdges.position(area.Bottom+1, e.Rows))*table.H
	return ui.Rect{X: left, Y: top, W: right - left, H: bottom - top}
}

// cellAt 返回指针所在的普通或合并单元格
func (e *item) cellAt(box ui.Rect, x, y float32) cellRange {
	row := e.RowEdges.cellAt(float64((y-box.Y)/box.H), e.Rows)
	column := e.ColumnEdges.cellAt(float64((x-box.X)/box.W), e.Columns)
	cell := cellRange{Top: row, Left: column, Bottom: row, Right: column}
	for _, area := range e.Merges {
		if area.overlaps(cell) {
			return area
		}
	}
	return cell
}

// tableDividerAt 返回指针附近可见的内部行列分隔线
func (e *item) tableDividerAt(box ui.Rect, x, y float32) (row, column int) {
	if !box.Contains(x, y) {
		return
	}
	area := e.cellAt(box, x, y)
	cell := e.rangeBox(box, area)
	if area.Left > 0 && math.Abs(float64(x-cell.X)) <= handleSize/2 {
		column = area.Left
	} else if area.Right+1 < e.Columns && math.Abs(float64(x-cell.X-cell.W)) <= handleSize/2 {
		column = area.Right + 1
	}
	if area.Top > 0 && math.Abs(float64(y-cell.Y)) <= handleSize/2 {
		row = area.Top
	} else if area.Bottom+1 < e.Rows && math.Abs(float64(y-cell.Y-cell.H)) <= handleSize/2 {
		row = area.Bottom + 1
	}
	return
}

// tableResizeCursor 在表格分隔线上显示对应方向的拖动光标
func (e *editor) tableResizeCursor(canvas ui.Element, paper ui.Rect) {
	a := e.selectedItem()
	if a == nil || a.Kind != kindTable || e.inline {
		return
	}
	row, column := 0, 0
	if e.drag != nil {
		row, column = e.drag.tableRow, e.drag.tableColumn
	} else if x, y, over := canvas.PointerPosition(); over {
		row, column = a.tableDividerAt(a.box(e.label, paper), x, y)
	}
	switch {
	case row > 0 && column > 0:
		canvas.Cursor(ui.CursorMove)
	case row > 0:
		canvas.Cursor(ui.CursorResizeRow)
	case column > 0:
		canvas.Cursor(ui.CursorResizeColumn)
	}
}

// tableInterior 返回表格内可选单元格的范围
func tableInterior(box ui.Rect) ui.Rect {
	inset := min(4, min(box.W, box.H)/4)
	return ui.Rect{X: box.X + inset, Y: box.Y + inset, W: box.W - 2*inset, H: box.H - 2*inset}
}

// selectTableCells 开始单元格拖选或双击编辑
func (e *editor) selectTableCells(a *item, box ui.Rect, ev ui.InputEvent) {
	cell := a.cellAt(box, ev.X, ev.Y)
	previous := e.cells
	e.cells = &tableSelection{start: cell, end: cell}
	if previous != nil {
		if ev.Mods&ui.Shift != 0 {
			e.cells.start = previous.start
		} else if ev.Clicks == 2 && previous.bounds(a) == cell {
			e.inline = true
			return
		}
	}
	e.drag = &gesture{selectCells: true, previousCells: previous}
}

// mergeCells 合并选区并替换其中原有的合并区域
func (e *item) mergeCells(area cellRange) {
	merges := slices.DeleteFunc(slices.Clone(e.Merges), area.overlaps)
	e.Merges = append(merges, area)
}

// cellValue 返回单元格文字及其排版设置
func (e *item) cellValue(row, column int) labelText {
	if row < len(e.Cells) && column < len(e.Cells[row]) {
		if cell := e.Cells[row][column]; cell != nil {
			return labelText(*cell)
		}
	}
	return labelText{FontSize: tableFontSize, BlockAlign: ui.Center}
}

// setCell 替换单元格文字并保留已保存快照的数据
func (e *item) setCell(row, column int, value labelText) {
	rows := make([][]*tableCell, max(len(e.Cells), row+1))
	copy(rows, e.Cells)
	cells := make([]*tableCell, max(len(rows[row]), column+1))
	copy(cells, rows[row])
	cell := tableCell(value)
	cells[column] = &cell
	rows[row] = cells
	e.Cells = rows
}

// tableTextBox 返回留出左右边距的单元格文字区域
func tableTextBox(rect ui.Rect, unitsPerMM float32) ui.Rect {
	inset := min(tableTextInset*unitsPerMM, rect.W/4)
	rect.X += inset
	rect.W -= 2 * inset
	return rect
}

// paintTable 绘制合并后的单元格文字和表格边界
func (e *item) paintTable(p *ui.Painter, rect ui.Rect, unitsPerMM float32, ink ui.Color) {
	width := float32(e.Stroke) * unitsPerMM
	var merges [maximumTableCount][maximumTableCount]*cellRange
	for i := range e.Merges {
		area := &e.Merges[i]
		for row := area.Top; row <= area.Bottom; row++ {
			for column := area.Left; column <= area.Right; column++ {
				merges[row][column] = area
			}
		}
	}
	for row := range e.Rows {
		for column := range e.Columns {
			area := cellRange{Top: row, Left: column, Bottom: row, Right: column}
			if merged := merges[row][column]; merged != nil {
				if row != merged.Top || column != merged.Left {
					continue
				}
				area = *merged
			}
			cell := e.rangeBox(rect, area)
			if value := e.cellValue(row, column); value.Value != "" || value.RichText != nil {
				value.paintText(p, tableTextBox(cell, unitsPerMM), unitsPerMM, ink)
			}
			right, bottom := cell.X+cell.W, cell.Y+cell.H
			if area.Bottom+1 < e.Rows {
				p.Line(cell.X, bottom, right, bottom, width, ink)
			}
			if area.Right+1 < e.Columns {
				p.Line(right, cell.Y, right, bottom, width, ink)
			}
		}
	}
	p.Stroke(rect, ink, 0, width)
}

// tableProperties 提供表格结构设置和所选单元格的富文本设置
func (e *editor) tableProperties(c *ui.Context, a *item) {
	rows, columns, stroke := a.Rows, a.Columns, a.Stroke
	minRows, minColumns := 1, 1
	for _, area := range a.Merges {
		minRows = max(minRows, area.Bottom+1)
		minColumns = max(minColumns, area.Right+1)
	}
	ui.Row(c.Key("counts")).Gap(8).Children(func() {
		for _, field := range []struct {
			name  string
			count *int
			low   int
		}{{i18n.Tr("table.rows"), &a.Rows, minRows}, {i18n.Tr("table.columns"), &a.Columns, minColumns}} {
			field := field
			ui.Column(c.Key(field.name)).Grow(1).MinWidth(0).Gap(5).Children(func() {
				ui.Text(c, field.name).FontSize(10).TextColor(colors.Muted)
				value := float64(*field.count)
				inspectorNumber(c.Key("count"), field.name, "", 0, &value, float64(field.low), maximumTableCount, 0, false)
				*field.count = int(value)
			})
		}
	})
	ui.Column(c.Key("stroke")).Gap(5).Children(func() {
		ui.Text(c, i18n.Tr("table.stroke")).FontSize(10).TextColor(colors.Muted)
		inspectorNumber(c.Key("stroke"), i18n.Tr("table.stroke"), "mm", 0, &a.Stroke, minimumLineWidth, maximumTableStroke, 1, false)
	})
	if rows != a.Rows {
		a.RowEdges = nil
	}
	if columns != a.Columns {
		a.ColumnEdges = nil
	}
	if rows != a.Rows || columns != a.Columns || stroke != a.Stroke {
		e.changed()
	}

	var area cellRange
	if e.cells != nil {
		area = e.cells.bounds(a)
		if area.Bottom >= a.Rows || area.Right >= a.Columns {
			e.cells = nil
		}
	}
	index := slices.Index(a.Merges, area)
	title := i18n.Tr("table.merge")
	if index >= 0 {
		title = i18n.Tr("table.unmerge")
	}
	if secondaryButton(c.Key("merge-cells"), title, e.cells != nil && area.multiple()) {
		if index >= 0 {
			a.Merges = slices.Delete(slices.Clone(a.Merges), index, index+1)
		} else {
			a.mergeCells(area)
		}
		e.changed()
	}
	if e.cells == nil || (area.multiple() && !slices.Contains(a.Merges, area)) {
		e.flushTextEdit()
		e.textEdit = nil
		return
	}
	cell := e.editTableCell(a, area)
	if e.textProperties(c.Key("cell-text").Key(area.Top).Key(area.Left), &cell) {
		a.setCell(area.Top, area.Left, cell)
		e.changed()
	}
}

// editTableCell 切换富文本会话并返回选中单元格的排版设置
func (e *editor) editTableCell(a *item, area cellRange) labelText {
	row, column := area.Top, area.Left
	cell := a.cellValue(row, column)
	if e.textEdit == nil || e.textEdit.row != row || e.textEdit.column != column {
		e.flushTextEdit()
		e.textEdit = newTextEdit(cell)
		e.textEdit.row, e.textEdit.column = row, column
	}
	return cell
}
