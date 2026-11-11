package editor

import (
	"cmp"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
	"github.com/go-text/typesetting/segmenter"
	"github.com/pengh/GoLabel/internal/i18n"
)

// millimetersPerPoint 是一个排版点对应的毫米数
const millimetersPerPoint = 25.4 / 72

// labelText 保存文字内容、字号和排版属性
type labelText struct {
	Value      string         `json:"value,omitempty"`
	FontSize   float64        `json:"fontMm,omitempty"`
	RichText   *richtext.Text `json:"richText,omitempty"`
	Vertical   bool           `json:"vertical,omitempty"`
	BlockAlign ui.Align       `json:"blockAlign,omitempty"`
}

// textEditState 保存画布和属性栏共用的富文本文档及已写回内容
type textEditState struct {
	doc         *richtext.Document
	written     richtext.Text
	row, column int
}

// newTextEdit 根据已有文字创建编辑会话
func newTextEdit(text labelText) *textEditState {
	content := text.textContent()
	return &textEditState{doc: richtext.NewDocument(content), written: content}
}

// writeTo 将富文本编辑结果写入文字内容
func (s *textEditState) writeTo(element *labelText) bool {
	content := s.doc.Text()
	if content == s.written {
		return false
	}
	element.RichText, element.Value, s.written = &content, "", content
	return true
}

// flushTextEdit 在切换目标前写回当前文字编辑
func (e *editor) flushTextEdit() {
	state := e.textEdit
	if state == nil {
		return
	}
	a := &e.label.Items[e.selected]
	if a.Kind == kindTable {
		cell := a.cellValue(state.row, state.column)
		if state.writeTo(&cell) {
			a.setCell(state.row, state.column, cell)
			e.changed()
		}
	} else if state.writeTo(&a.labelText) {
		e.changed()
	}
}

// selectItem 切换选中的元素并保留已输入的文字
func (e *editor) selectItem(index int) {
	e.inline = false
	if index == e.selected {
		return
	}
	e.flushTextEdit()
	e.selected, e.textEdit, e.cells = index, nil, nil
	if index >= 0 && e.label.Items[index].Kind == kindText {
		e.textEdit = newTextEdit(e.label.Items[index].labelText)
	}
}

// inlineEditor 在选中的文字框或表格单元格内编辑文字
func (e *editor) inlineEditor(c *ui.Context, canvas ui.Element) {
	w, h := c.Size()
	size := ui.Point{X: w, Y: h}
	lastSize := ui.Local(canvas, "window-size", func() ui.Point { return size })
	if *lastSize != size {
		*lastSize = size
		c.After(0)
	}
	bounds := canvas.Bounds()
	paper := e.paperRect(ui.Rect{W: bounds.W, H: bounds.H})
	element := &e.label.Items[e.selected]
	rect := element.box(e.label, paper)
	unitsPerMM := paper.H / float32(e.label.Height)
	content, name := element.labelText, i18n.Tr("text.canvas")
	if element.Kind == kindTable {
		area := e.cells.bounds(element)
		content = e.editTableCell(element, area)
		rect = tableTextBox(element.rangeBox(rect, area), unitsPerMM)
		name = i18n.Tr("text.cell")
	}
	scale := unitsPerMM * millimetersPerPoint
	richtext.EditorBase(c, e.textEdit.doc, richtext.EditorOptions{
		Label: name, Font: ui.Font{Size: float32(content.FontSize / millimetersPerPoint)}, NoMarkdown: true,
		Theme: &richtext.Theme{Text: ui.RGB(0, 0, 0)},
	}).Absolute().Left(rect.X).Top(rect.Y).Size(rect.W/scale, rect.H/scale).
		Scale(scale, scale).TransformOrigin(0, 0).Background(ui.RGB(255, 255, 255)).FocusRing(false).AutoFocus()
}

// textContent 返回已有富文本或将纯文字转换为可编辑内容
func (e *labelText) textContent() richtext.Text {
	if e.RichText != nil {
		return *e.RichText
	}
	return richtext.NewText(e.Value)
}

// formatText 对选区应用格式，未选择文字时应用到整个文字框
func formatText(doc *richtext.Document, format func()) {
	anchor, caret := doc.Selection()
	if anchor != caret || doc.Text().Len() == 0 {
		format()
		return
	}
	doc.SetSelection(0, doc.Text().Len())
	format()
	doc.SetSelection(anchor, caret)
}

// horizontalAlignIcons 是左对齐、水平居中和右对齐图标
var horizontalAlignIcons = [...]*ui.SVG{
	propertyIcon("M4 5h16M4 10h10M4 14h16M4 19h10"),
	propertyIcon("M4 5h16M7 10h10M4 14h16M7 19h10"),
	propertyIcon("M4 5h16m-10 5h10M4 14h16m-10 5h10"),
}

// verticalAlignIcons 是上对齐、垂直居中和下对齐图标
var verticalAlignIcons = [...]*ui.SVG{
	propertyIcon("M4 4h16m-8 5v11m-5-6 5-5 5 5"),
	propertyIcon("M4 12h16M12 3v5M9 6l3 3 3-3m-3 10v5m-3-3 3-3 3 3"),
	propertyIcon("M4 20h16m-8-5V4m-5 6 5 5 5-5"),
}

// propertyIcon 创建由原生界面着色的属性栏图标
func propertyIcon(path string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" d="` + path + `"/></svg>`))
}

// propertyButton 显示带提示和选中态的紧凑图标按钮
func propertyButton(c *ui.Context, label string, icon *ui.SVG, selected bool) bool {
	button := ui.ButtonBase(c.Key(label)).Size(30, 30).Radius(6).Label(label).Tooltip(label).
		Role(ui.RoleButton).AccessSelected(selected)
	color := colors.Body2
	if selected {
		button.Background(colors.ChipSelected).Shadow(0, 1, 2, 0, colors.ShadowSoft)
		color = colors.Accent
	} else if button.Hovered() {
		button.Background(colors.ChipPressed)
	}
	button.Children(func() { ui.Icon(c, icon).Size(15, 15).TextColor(color) })
	return button.Clicked()
}

// alignmentGroup 显示单轴的三个对齐图标并更新所选值
func alignmentGroup(c *ui.Context, labels [3]string, icons [3]*ui.SVG, value *ui.Align) {
	ui.Row(c).Grow(1).Padding(2).Gap(2).Justify(ui.SpaceBetween).Radius(7).Children(func() {
		for i, label := range labels {
			if propertyButton(c, label, icons[i], *value == ui.Align(i)) {
				*value = ui.Align(i)
			}
		}
	})
}

// textProperties 构建文字内容、字体、字形和对齐设置
func (e *editor) textProperties(c *ui.Context, element *labelText) bool {
	if e.textEdit == nil {
		e.textEdit = newTextEdit(*element)
	}
	panel := ui.Column(c.Key("text-format")).FillWidth().Gap(8)
	state := e.textEdit
	doc := state.doc
	changed := false
	panel.Children(func() {
		font := ui.Font{Size: float32(element.FontSize / millimetersPerPoint)}
		sectionHeading(c, i18n.Tr("text.heading"))
		if !e.inline {
			richtext.EditorBase(c.Key("text"), doc, richtext.EditorOptions{
				Label: i18n.Tr("text.body"), Placeholder: i18n.Tr("text.placeholder"), Font: font, NoMarkdown: true,
				Theme: &richtext.Theme{Padding: [2]float32{8, 9}},
			}).FillWidth().Height(66).Shrink(0).Radius(8).Border(1, colors.FieldBorder).
				Background(colors.FieldBg)
		}

		hairline(c.Key("divider"))

		formats := doc.Formatting()
		fonts := labelFonts()
		name := cmp.Or(fonts.nameByFamily[formats.Attributes.Font], formats.Attributes.Font)
		fontDownloadHeading(c.Key("font-heading"))
		ui.Row(c.Key("font-row")).Gap(8).Children(func() {
			if ui.Combobox(c.Key("font"), &name, fonts.options).Grow(1).Height(34).Radius(7).
				FontSize(12).Label(i18n.Tr("text.font")).Changed() {
				family := fonts.familyByName[name]
				formatText(doc, func() { doc.Format(func(a *richtext.Attributes) { a.Font = family }) })
			}
			size := float64(formats.Attributes.Size)
			if size == 0 {
				size = float64(font.Size)
			}
			before := size
			inspectorNumber(c.Key("font-size"), i18n.Tr("text.size"), "pt", 108, &size, 1, 114, 1, true)
			if size != before {
				formatText(doc, func() { doc.Format(func(a *richtext.Attributes) { a.Size = float32(size) }) })
			}
		})

		horizontal, vertical := formats.Paragraph.Align, element.BlockAlign
		if element.Vertical {
			horizontal, vertical = ui.End-element.BlockAlign, formats.Paragraph.Align
		}
		wasVertical := element.Vertical
		direction := 0
		if wasVertical {
			direction = 1
		}

		ui.Row(c.Key("styles")).Gap(2).AlignItems(ui.Center).Children(func() {
			for i, style := range []struct {
				title  string
				glyph  string
				state  richtext.FormatState
				toggle func()
			}{
				{i18n.Tr("text.bold"), "B", formats.Bold, doc.ToggleBold},
				{i18n.Tr("text.italic"), "I", formats.Italic, doc.ToggleItalic},
				{i18n.Tr("text.underline"), "U", formats.Underline, doc.ToggleUnderline},
				{i18n.Tr("text.strike"), "S", formats.Strikethrough, doc.ToggleStrikethrough},
			} {
				button := ui.ButtonBase(c.Key(style.title)).Size(30, 30).Radius(6).Role(ui.RoleButton).
					Label(style.title).Tooltip(style.title).AccessSelected(style.state == richtext.FormatOn)
				ink := colors.Body2
				if style.state == richtext.FormatOn {
					button.Background(colors.Segmented).Shadow(0, 1, 2, 0, colors.ShadowSoft)
					ink = c.Theme().Accent
				} else if button.Hovered() {
					button.Background(colors.ChipPressed)
				}
				button.Children(func() {
					text := ui.Text(c, style.glyph).FontSize(12.5).FontWeight(600).TextColor(ink)
					switch i {
					case 1:
						text.Italic()
					case 2:
						text.Underline()
					case 3:
						text.Strikethrough()
					}
				})
				if button.Clicked() {
					formatText(doc, style.toggle)
				}
			}
			ui.Box(c.Key("spacer")).Grow(1)
			for i, label := range [3]string{i18n.Tr("text.alignLeft"), i18n.Tr("text.alignCenter"), i18n.Tr("text.alignRight")} {
				if propertyButton(c, label, horizontalAlignIcons[i], horizontal == ui.Align(i)) {
					horizontal = ui.Align(i)
				}
			}
		})

		ui.Row(c.Key("alignment")).Gap(8).Children(func() {
			alignmentGroup(c.Key("vertical"), [3]string{i18n.Tr("text.alignTop"), i18n.Tr("text.alignMiddle"), i18n.Tr("text.alignBottom")}, verticalAlignIcons, &vertical)
			ui.Column(c.Key("direction")).Width(128).Shrink(0).Children(func() {
				optionRow(c.Key("options"), []string{i18n.Tr("text.horizontal"), i18n.Tr("text.vertical")}, direction, func(i int) {
					direction = i
				})
			})
		})

		if vertical := direction == 1; vertical != wasVertical {
			element.Vertical = vertical
			changed = true
		}
		inline, block := horizontal, vertical
		if element.Vertical {
			inline, block = vertical, ui.End-horizontal
		}
		if inline != formats.Paragraph.Align {
			formatText(doc, func() { doc.SetAlign(inline) })
		}
		if block != element.BlockAlign {
			element.BlockAlign = block
			changed = true
		}
	})
	return state.writeTo(element) || changed
}

// textSpan 将富文本的字符样式转换为当前画布比例的排版样式
func textSpan(run richtext.Run, pointScale float32) ui.Span {
	a := run.Attributes
	return ui.Span{
		Text: run.Text, Font: a.Font, Size: a.Size * pointScale, Weight: a.Weight,
		Italic: a.Italic, Underline: a.Underline, Strikethrough: a.Strikethrough,
		Color: a.Color, Background: a.Background,
	}
}

// alignmentOffset 计算内容在给定空间内按起点、居中或末端对齐的偏移
func alignmentOffset(available, used float32, align ui.Align) float32 {
	space := max(0, available-used)
	switch align {
	case ui.Center:
		return space / 2
	case ui.End:
		return space
	default:
		return 0
	}
}

// paintText 在文字区域内绘制横排或竖排内容并应用双轴对齐
func (e *labelText) paintText(p *ui.Painter, rect ui.Rect, unitsPerMM float32, ink ui.Color) {
	font := ui.Font{Size: float32(e.FontSize) * unitsPerMM}
	p.Clip(rect, 0, func() {
		if e.RichText == nil && !e.Vertical {
			if e.BlockAlign == ui.Start {
				p.Text(rect.X, rect.Y, e.Value, font.Size, ink)
				return
			}
			layout := ui.ShapeText(e.Value, font, 0)
			_, height := layout.Size()
			p.TextLayout(layout, rect.X, rect.Y+alignmentOffset(rect.H, height, e.BlockAlign), ink)
			return
		}
		pointScale := unitsPerMM * millimetersPerPoint
		text := e.textContent()
		if e.Vertical {
			paintVerticalText(p, rect, text, font, pointScale, ink, e.BlockAlign)
			return
		}

		var layouts []*ui.TextLayout
		var totalHeight float32
		for _, para := range text.Paragraphs() {
			spans := make([]ui.Span, len(para.Runs))
			for i, run := range para.Runs {
				spans[i] = textSpan(run, pointScale)
			}
			layout := ui.ShapeRichTextWith(spans, font, ui.TextShape{Width: rect.W, Align: para.Style.Align})
			layouts = append(layouts, layout)
			_, height := layout.Size()
			totalHeight += height
			if totalHeight >= rect.H {
				break
			}
		}
		y := rect.Y + alignmentOffset(rect.H, totalHeight, e.BlockAlign)
		for _, layout := range layouts {
			p.TextLayout(layout, rect.X, y, ink)
			_, height := layout.Size()
			y += height
		}
	})
}

// verticalGlyph 保存竖排中一个字素的排版结果和占用高度
type verticalGlyph struct {
	layout *ui.TextLayout
	width  float32
	height float32
}

// verticalColumn 保存竖排的一列字素及该列的宽高和纵向对齐
type verticalColumn struct {
	glyphs        []verticalGlyph
	width, height float32
	align         ui.Align
}

// paintVerticalText 将字素自上而下排列并让整个列组水平对齐
func paintVerticalText(p *ui.Painter, rect ui.Rect, text richtext.Text, font ui.Font, pointScale float32, ink ui.Color, block ui.Align) {
	font.Features = "vert"
	var seg segmenter.Segmenter
	var columns []verticalColumn
	var totalWidth float32
	addColumn := func(column verticalColumn) {
		columns = append(columns, column)
		totalWidth += column.width
	}

paragraphs:
	for _, para := range text.Paragraphs() {
		column := verticalColumn{width: font.Size, align: para.Style.Align}
		for _, run := range para.Runs {
			span := textSpan(run, pointScale)
			seg.Init([]rune(run.Text))
			for it := seg.GraphemeIterator(); it.Next(); {
				span.Text = string(it.Grapheme().Text)
				layout := ui.ShapeRichText([]ui.Span{span}, font, 0)
				w, h := layout.Size()
				if len(column.glyphs) > 0 && column.height+h > rect.H {
					addColumn(column)
					if totalWidth >= rect.W {
						break paragraphs
					}
					column = verticalColumn{width: font.Size, align: para.Style.Align}
				}
				column.glyphs = append(column.glyphs, verticalGlyph{layout: layout, width: w, height: h})
				column.width, column.height = max(column.width, w), column.height+h
			}
		}
		addColumn(column)
		if totalWidth >= rect.W {
			break
		}
	}
	right := rect.X + rect.W - alignmentOffset(rect.W, totalWidth, block)
	for _, column := range columns {
		right -= column.width
		y := rect.Y + alignmentOffset(rect.H, column.height, column.align)
		for _, glyph := range column.glyphs {
			p.TextLayout(glyph.layout, right+(column.width-glyph.width)/2, y, ink)
			y += glyph.height
		}
	}
}
