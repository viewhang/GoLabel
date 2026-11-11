package editor

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

const (
	// templatePreviewWidth 是模板库右侧预览面板的宽度
	templatePreviewWidth = 280
	// templatePreviewHeight 是预览纸张区的高度
	templatePreviewHeight = 172
	// templateCardHeight 是模板卡片的固定高度
	templateCardHeight = 112
	// templateThumbHeight 是卡片缩略图区的高度
	templateThumbHeight = 68
	// templateThumbnailSize 是用户模板缩略图最长边的像素数
	templateThumbnailSize = 640
)

// labelTemplate 保存模板名称、分类、规格和内置示例或用户标签
type labelTemplate struct {
	name      string
	category  string
	width     float64
	height    float64
	dpi       int
	zhItems   []item
	enItems   []item
	custom    *label
	thumbnail *ui.Bitmap
}

// label 生成可独立编辑的标签，内置示例内容跟随当前界面语言
func (t labelTemplate) label() *label {
	if t.custom != nil {
		copy := *t.custom
		copy.Items = cloneItems(copy.Items)
		return &copy
	}
	items := t.zhItems
	if i18n.Locale() == string(languageEnglish) {
		items = t.enItems
	}
	return &label{Width: t.width, Height: t.height, DPI: t.dpi, Items: cloneItems(items)}
}

// cloneItems 深拷贝模板元素，避免多个文档共用同一份数据
func cloneItems(items []item) []item {
	data, err := json.Marshal(items)
	if err != nil {
		return slices.Clone(items)
	}
	var copy []item
	if err := json.Unmarshal(data, &copy); err != nil {
		return slices.Clone(items)
	}
	for i := range copy {
		copy[i].bitmap = items[i].bitmap
	}
	return copy
}

// tplText 构造模板中的文字元素，size 为毫米字号
func tplText(x, y, w, h, size float64, value string, align ui.Align) item {
	return item{
		Kind: kindText, X: x, Y: y, W: w, H: h,
		labelText: labelText{Value: value, FontSize: size, BlockAlign: align},
	}
}

// tplLine 构造模板中的分隔线
func tplLine(x, y, w float64) item {
	return item{Kind: kindLine, X: x, Y: y, W: w, H: 0.4}
}

// tplBarcode 构造模板中的 Code 128 条码
func tplBarcode(x, y, w, h float64, value string) item {
	return item{Kind: kindBarcode, X: x, Y: y, W: w, H: h, labelText: labelText{Value: value}}
}

// tplQR 构造模板中的二维码
func tplQR(x, y, side float64, value string) item {
	return item{Kind: kindQR, X: x, Y: y, W: side, H: side, labelText: labelText{Value: value}}
}

// templateCategories 是模板库顶部的分类筛选，第一项为全部
var templateCategories = [...]string{"all", "product", "food", "retail", "shipping", "asset", "custom"}

// categoryName 返回模板分类在当前语言下的名称
func categoryName(category string) string {
	return i18n.Tr("template.category." + category)
}

// title 返回用户输入的名称或内置模板的本地化名称
func (t labelTemplate) title() string {
	if t.custom != nil {
		return t.name
	}
	return i18n.Tr(t.name)
}

// searchText 返回可供搜索的用户模板名称或内置模板的中英文文案
func (t labelTemplate) searchText() string {
	if t.custom != nil {
		return strings.ToLower(t.name)
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{t.name, "template.category." + t.category} {
		if message, ok := i18n.Lookup(key); ok {
			parts = append(parts, message.ZH, message.EN)
		}
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// labelTemplates 保存内置模板和从本机读取或新保存的用户模板
var labelTemplates = []labelTemplate{
	{
		name: "template.product", category: "product", width: 50, height: 30, dpi: 203,
		zhItems: []item{
			tplText(24, 2, 23, 4, 2.8, "山茶手作", ui.End),
			tplText(3, 6, 34, 10, 8, "桂花乌龙", ui.Start),
			tplText(3.2, 16.5, 34, 3, 2.2, "OSMANTHUS OOLONG", ui.Start),
			tplLine(3, 19.6, 28),
			tplText(3, 20.6, 20, 4, 2.8, "净含量 50g", ui.Start),
			tplBarcode(26, 20.6, 21, 7.4, "6901234567890"),
		},
		enItems: []item{
			tplText(24, 2, 23, 4, 2.8, "CAMELLIA", ui.End),
			tplText(3, 7, 34, 9, 6, "Oolong Tea", ui.Start),
			tplText(3.2, 16.2, 34, 3, 2.2, "OSMANTHUS OOLONG", ui.Start),
			tplLine(3, 19.6, 28),
			tplText(3, 20.6, 20, 4, 2.8, "Net 50 g", ui.Start),
			tplBarcode(26, 20.6, 21, 7.4, "6901234567890"),
		},
	},
	{
		name: "template.food", category: "food", width: 60, height: 40, dpi: 203,
		zhItems: []item{
			tplLine(3, 2.6, 54),
			tplText(3, 5, 54, 10, 8.6, "草莓果酱", ui.Center),
			tplText(3, 15.6, 54, 3, 2.3, "STRAWBERRY JAM", ui.Center),
			tplLine(3, 19.4, 54),
			tplText(4, 21, 52, 4, 2.5, "草莓 · 白砂糖 · 柠檬汁", ui.Center),
			tplText(4, 26, 52, 4.4, 2.8, "净含量 200g", ui.Center),
			tplLine(3, 37.4, 54),
		},
		enItems: []item{
			tplLine(3, 2.6, 54),
			tplText(3, 6, 54, 9, 6.4, "Strawberry Jam", ui.Center),
			tplText(3, 15.6, 54, 3, 2.3, "STRAWBERRY JAM", ui.Center),
			tplLine(3, 19.4, 54),
			tplText(4, 21, 52, 4, 2.4, "Strawberry · sugar · lemon", ui.Center),
			tplText(4, 26, 52, 4.4, 2.8, "Net 200 g", ui.Center),
			tplLine(3, 37.4, 54),
		},
	},
	{
		name: "template.price", category: "retail", width: 40, height: 30, dpi: 203,
		zhItems: []item{
			tplText(3, 2.6, 22, 4, 2.8, "今日特价", ui.Start),
			tplText(3, 6.6, 26, 14, 12, "¥39", ui.Start),
			tplText(3, 21, 24, 4, 3.2, "特级初榨", ui.Start),
			tplText(3, 25.4, 16, 3.6, 2.4, "500ml", ui.Start),
			tplText(22, 25.4, 15, 3.6, 2.4, "意大利", ui.End),
		},
		enItems: []item{
			tplText(3, 2.6, 26, 4, 2.6, "TODAY ONLY", ui.Start),
			tplText(3, 6.6, 26, 14, 12, "¥39", ui.Start),
			tplText(3, 21, 24, 4, 2.8, "Extra virgin", ui.Start),
			tplText(3, 25.4, 16, 3.6, 2.4, "500 ml", ui.Start),
			tplText(22, 25.4, 15, 3.6, 2.4, "Italy", ui.End),
		},
	},
	{
		name: "template.shipping", category: "shipping", width: 100, height: 50, dpi: 203,
		zhItems: []item{
			tplText(4, 3.4, 40, 5, 3.4, "STANDARD", ui.Start),
			tplLine(4, 9.6, 92),
			tplText(4, 12, 20, 4, 3, "收件人", ui.Start),
			tplText(4, 18, 44, 6, 2.6, "地址", ui.Start),
			tplText(4, 26, 44, 4, 2.6, "订单 GL-2048", ui.Start),
			tplLine(4, 33, 44),
			tplBarcode(52, 12, 44, 18, "GL2048001"),
			tplText(52, 31, 44, 3.4, 2.2, "GL2048001", ui.Center),
		},
		enItems: []item{
			tplText(4, 3.4, 40, 5, 3.4, "STANDARD", ui.Start),
			tplLine(4, 9.6, 92),
			tplText(4, 12, 20, 4, 3, "Ship to", ui.Start),
			tplText(4, 18, 44, 6, 2.6, "Address", ui.Start),
			tplText(4, 26, 44, 4, 2.6, "Order GL-2048", ui.Start),
			tplLine(4, 33, 44),
			tplBarcode(52, 12, 44, 18, "GL2048001"),
			tplText(52, 31, 44, 3.4, 2.2, "GL2048001", ui.Center),
		},
	},
	{
		name: "template.asset", category: "asset", width: 50, height: 20, dpi: 203,
		zhItems: []item{
			tplText(3, 2.4, 24, 3.4, 2.4, "资产编号", ui.Start),
			tplText(3, 5.6, 30, 7.4, 5.6, "GL-0042", ui.Start),
			tplText(3, 13.4, 34, 3.6, 2.3, "MacBook Pro · 设计部", ui.Start),
			tplText(34, 13.4, 13, 3.6, 2.3, "2026.01", ui.End),
		},
		enItems: []item{
			tplText(3, 2.4, 24, 3.4, 2.4, "ASSET ID", ui.Start),
			tplText(3, 5.6, 30, 7.4, 5.6, "GL-0042", ui.Start),
			tplText(3, 13.4, 34, 3.6, 2.3, "MacBook Pro · Design", ui.Start),
			tplText(34, 13.4, 13, 3.6, 2.3, "2026.01", ui.End),
		},
	},
	{
		name: "template.qr", category: "product", width: 40, height: 40, dpi: 203,
		zhItems: []item{
			tplText(3, 2.6, 34, 4.6, 3.2, "扫码查看", ui.Start),
			tplQR(9, 8.4, 22, "https://golabel.example.com/p/42"),
			tplText(3, 32.4, 34, 3.6, 2.4, "产品信息", ui.Center),
		},
		enItems: []item{
			tplText(3, 2.8, 34, 4.6, 2.8, "Scan for details", ui.Start),
			tplQR(9, 8.4, 22, "https://golabel.example.com/p/42"),
			tplText(3, 32.4, 34, 3.6, 2.4, "Product info", ui.Center),
		},
	},
}

// categoryColor 返回模板分类对应的标识色
func categoryColor(category string) ui.Color {
	switch category {
	case "product":
		return colors.Accent
	case "food":
		return ui.Hex("#12B76A")
	case "retail":
		return ui.Hex("#F79009")
	case "shipping":
		return ui.Hex("#0BA5EC")
	case "asset":
		return ui.Hex("#7A5AF8")
	default:
		return colors.Muted
	}
}

// visibleTemplates 返回符合搜索和分类条件的模板下标
func (e *editor) visibleTemplates() []int {
	query := strings.ToLower(strings.TrimSpace(e.settings.query))
	var indexes []int
	for i, t := range labelTemplates {
		if e.settings.category > 0 && t.category != templateCategories[e.settings.category] {
			continue
		}
		if query != "" && !strings.Contains(t.searchText(), query) {
			continue
		}
		indexes = append(indexes, i)
	}
	return indexes
}

// useTemplate 用选中模板新建一个标签
func (e *editor) useTemplate(t labelTemplate) {
	e.settings.open = false
	l := t.label()
	l.title = t.title()
	if e.tabs == nil {
		e.label, e.selected, e.drag = l, -1, nil
		e.textEdit, e.cells, e.inline = nil, nil, false
		for i := range l.Items {
			element := &l.Items[i]
			if e.graphics != nil && (element.Kind == kindQR || element.Kind == kindBarcode) {
				element.prepareSymbol(e.graphics)
			}
		}
		return
	}
	e.tabs.newTab(l)
}

// templatePage 在内容区剩余高度中显示模板列表和选中模板预览
func (e *editor) templatePage(c *ui.Context) {
	ui.Row(c).Grow(1).MinHeight(0).Gap(20).AlignItems(ui.Stretch).Children(func() {
		if e.templateList(c.Key("list")) {
			e.templateDetail(c.Key("detail"), labelTemplates[e.settings.template])
		} else {
			ui.Box(c.Key("detail")).Width(templatePreviewWidth).Shrink(0)
		}
	})
}

// templateList 显示搜索和模板卡片，并报告筛选后是否有可用模板
func (e *editor) templateList(c *ui.Context) bool {
	var indexes []int
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
		ui.Text(c.Key("title"), i18n.Tr("template.library")).FontSize(20).FontWeight(650).TextColor(colors.Heading)
		e.templateSearch(c.Key("search"))
		e.templateChips(c.Key("chips"))
		indexes = e.visibleTemplates()
		if len(indexes) > 0 && !slices.Contains(indexes, e.settings.template) {
			e.settings.template = indexes[0]
		}
		ui.Scroll(c.Key("scroll")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key("grid")).Padding(0, 1, 4, 1).Gap(10).Children(func() {
				if len(indexes) == 0 {
					e.templateEmpty(c.Key("empty"))
					return
				}
				for i := 0; i < len(indexes); i += 2 {
					ui.Row(c.Key(i)).Gap(10).Children(func() {
						for _, index := range indexes[i:min(i+2, len(indexes))] {
							e.templateCard(c.Key(index), index)
						}
					})
				}
			})
		})
	})
	return len(indexes) > 0
}

// templateSearch 显示模板搜索输入框
func (e *editor) templateSearch(c *ui.Context) {
	ui.Row(c).Margin(14, 0, 0, 0).Height(36).Radius(8).Padding(0, 10).Gap(8).
		BorderWidth(1).BorderColor(colors.FieldBorder).Background(colors.FieldBg).
		AlignItems(ui.Center).Children(func() {
		ui.Icon(c.Key("icon"), iconSearch).Size(14, 14).Shrink(0).TextColor(colors.Faint)
		ui.TextInputBase(c.Key("entry"), &e.settings.query).Grow(1).MinWidth(0).Height(32).
			FontSize(12).Label(i18n.Tr("template.search")).Placeholder(i18n.Tr("template.search"))
	})
}

// templateChips 显示分类筛选按钮
func (e *editor) templateChips(c *ui.Context) {
	ui.Row(c).Margin(10, 0, 0, 0).Gap(6).Wrap().Children(func() {
		for i, category := range templateCategories {
			e.templateChip(c.Key(category), i, categoryName(category))
		}
	})
}

// templateChip 显示一个分类按钮，选中时主色描边
func (e *editor) templateChip(c *ui.Context, index int, name string) {
	selected := e.settings.category == index
	ink, border, fill := colors.Muted, colors.CardBorder, colors.Card
	if selected {
		ink, border, fill = colors.Text, colors.Accent, colors.AccentSoft
	}
	button := ui.ButtonBase(c).Height(28).Padding(0, 10).Radius(6).Shrink(0).
		Role(ui.RoleButton).Label(name).AccessSelected(selected).Cursor(ui.CursorPointer).
		BorderWidth(1).BorderColor(border).Background(fill)
	if button.Hovered() && !selected {
		button.Background(colors.Hover)
	}
	button.Children(func() {
		ui.Text(c, name).FontSize(10.5).FontWeight(map[bool]int{true: 600, false: 500}[selected]).
			TextColor(ink)
	})
	if button.Clicked() {
		e.settings.category = index
	}
}

// templateCard 显示一个模板缩略图和名称
func (e *editor) templateCard(c *ui.Context, index int) {
	t := labelTemplates[index]
	selected := e.settings.template == index
	ink, border, fill := colors.Text, colors.CardBorder, colors.Card
	if selected {
		ink, border, fill = colors.Accent, colors.Accent, colors.AccentSoft
	}
	width := float32(1)
	if selected {
		width = 1.5
	}
	card := ui.ButtonBase(c).Grow(1).Height(templateCardHeight).Padding(10).Gap(8).Column().
		AlignItems(ui.Stretch).Radius(12).Role(ui.RoleButton).Label(t.title()).
		AccessSelected(selected).Cursor(ui.CursorPointer).
		BorderWidth(width).BorderColor(border).Background(fill)
	if card.Hovered() && !selected {
		card.BorderColor(colors.BorderHover)
	}
	card.Children(func() {
		ui.Box(c.Key("thumb")).Height(templateThumbHeight).Radius(8).Clip().
			Draw(func(p *ui.Painter, r ui.Rect) { paintTemplateThumb(p, r, t) })
		ui.Row(c.Key("meta")).Gap(7).AlignItems(ui.Center).Children(func() {
			ui.Box(c.Key("dot")).Size(6, 6).Radius(3).Shrink(0).
				Background(categoryColor(t.category))
			ui.Text(c, t.title()).FontSize(12.5).FontWeight(600).SingleLine().Grow(1).
				TextColor(ink)
			ui.Textf(c, "%g × %g", t.width, t.height).FontSize(10).Shrink(0).
				TextColor(colors.Faint)
		})
	})
	if card.Clicked() {
		e.settings.template = index
	}
}

// templateEmpty 在没有匹配模板时提示修改筛选条件
func (e *editor) templateEmpty(c *ui.Context) {
	ui.Column(c).Margin(0, 14, 0, 0).Padding(30, 16).Radius(10).Gap(8).
		BorderWidth(1).BorderStyle(ui.BorderDashed).BorderColor(colors.DashBorder).
		Background(colors.FieldBg).AlignItems(ui.Center).Children(func() {
		ui.Icon(c.Key("icon"), iconSearch).Size(18, 18).TextColor(colors.RulerText)
		ui.Text(c, i18n.Tr("template.empty")).FontSize(11.5).FontWeight(600).TextColor(colors.Body2)
	})
}

// templateDetail 显示选中模板的预览、规格和使用按钮
func (e *editor) templateDetail(c *ui.Context, t labelTemplate) {
	ui.Column(c).Width(templatePreviewWidth).Shrink(0).FillHeight().Padding(12).Gap(0).
		Radius(12).BorderWidth(1).BorderColor(colors.Divider).
		Background(colors.PanelAlt).Children(func() {
		ui.Box(c.Key("preview")).Height(templatePreviewHeight).Radius(10).Clip().
			BorderWidth(1).BorderColor(colors.Divider).
			Draw(func(p *ui.Painter, r ui.Rect) { paintTemplatePreview(p, r, t) })

		ui.Column(c.Key("meta")).Margin(14, 0, 0, 0).Gap(4).Children(func() {
			ui.Text(c, t.title()).FontSize(16).FontWeight(650).TextColor(colors.Heading)
			ui.Textf(c, "%g × %g mm · %d DPI", t.width, t.height, t.dpi).FontSize(11).
				TextColor(colors.Muted)
		})

		ui.Box(c.Key("spacer")).Grow(1)
		hairline(c.Key("rule"))
		accentAction(c.Key("use"), i18n.Tr("template.use"), iconChevron, true, func() { e.useTemplate(t) })
	})
}

// previewLabel 返回用于绘制预览的模板内容
func (t labelTemplate) previewLabel() *label {
	if t.custom != nil {
		return t.custom
	}
	return t.label()
}

// paintDotField 绘制点阵底纹
func paintDotField(p *ui.Painter, r ui.Rect) {
	p.Fill(r, colors.Stage, 0)
	for y := r.Y + 7; y < r.Y+r.H; y += 14 {
		for x := r.X + 7; x < r.X+r.W; x += 14 {
			p.Fill(ui.Rect{X: x - 1, Y: y - 1, W: 2, H: 2}, colors.SketchDot, 1)
		}
	}
}

// paperIn 按标签比例在区域内居中放置纸张
func paperIn(r ui.Rect, l *label, inset float32) ui.Rect {
	scale := min((r.W-2*inset)/float32(l.Width), (r.H-2*inset)/float32(l.Height))
	w, h := float32(l.Width)*scale, float32(l.Height)*scale
	return ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}
}

// paintTemplateThumb 在卡片缩略图中绘制点阵底和标签示意
func paintTemplateThumb(p *ui.Painter, r ui.Rect, t labelTemplate) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	paintDotField(p, r)
	l := t.previewLabel()
	paper := paperIn(r, l, 10)
	p.Shadow(paper, 0, 0, 4, 10, -3, colors.ShadowMid)
	t.paintPreview(p, paper, l)
	p.Stroke(paper, ui.Hex("#D5DEE8"), 0, 1)
}

// paintTemplatePreview 在详情面板中绘制放大的标签示意
func paintTemplatePreview(p *ui.Painter, r ui.Rect, t labelTemplate) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	paintDotField(p, r)
	l := t.previewLabel()
	paper := paperIn(r, l, 16)
	p.Shadow(paper, 0, 0, 10, 24, -8, colors.ShadowStrong)
	t.paintPreview(p, paper, l)
	p.Stroke(paper, ui.Hex("#D5DEE8"), 0, 1)
}

// paintPreview 绘制用户模板的实际缩略图或内置模板示意
func (t labelTemplate) paintPreview(p *ui.Painter, paper ui.Rect, l *label) {
	if t.thumbnail != nil {
		p.Image(t.thumbnail, paper, ui.Contain)
		return
	}
	paintSketchLabel(p, paper, l)
}

// paintSketchLabel 按比例绘制标签示意，用图案代替需要图形上下文的码
func paintSketchLabel(p *ui.Painter, paper ui.Rect, l *label) {
	if paper.W <= 0 || paper.H <= 0 {
		return
	}
	ink := ui.Hex("#111827")
	// 纸张是印刷出来的白纸，深浅主题下都填白色
	p.Fill(paper, colors.Paper, 0)
	p.Clip(paper, 0, func() {
		sx, sy := paper.W/float32(l.Width), paper.H/float32(l.Height)
		for i := range l.Items {
			a := &l.Items[i]
			r := ui.Rect{
				X: paper.X + float32(a.X)*sx, Y: paper.Y + float32(a.Y)*sy,
				W: float32(a.W) * sx, H: float32(a.H) * sy,
			}
			switch a.Kind {
			case kindText:
				size := float32(a.FontSize) * sy
				if size < 1 {
					continue
				}
				layout := ui.ShapeText(a.Value, ui.Font{Size: size}, 0)
				width, height := layout.Size()
				x := r.X
				switch a.BlockAlign {
				case ui.Center:
					x = r.X + (r.W-width)/2
				case ui.End:
					x = r.X + r.W - width
				}
				p.Clip(r, 0, func() {
					p.TextLayout(layout, x, r.Y+(r.H-height)/2, ink)
				})
			case kindLine:
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2, W: r.W, H: max(1, r.H)}, ink, 0)
			case kindBarcode:
				paintSketchBarcode(p, r)
			case kindQR:
				paintSketchQR(p, r)
			case kindTable:
				paintSketchTable(p, r)
			case kindImage:
				p.Image(a.bitmap, r, ui.Contain)
			}
		}
	})
}

// paintSketchBarcode 用黑白竖条表示条码，底和纸一样是白色
func paintSketchBarcode(p *ui.Painter, r ui.Rect) {
	ink := ui.Hex("#111827")
	p.Fill(r, colors.Paper, 0)
	widths := [...]float32{1, 2, 1, 3, 1, 2, 2, 1, 3, 1, 2}
	unit := max(0.7, r.W/64)
	x := r.X + unit
	for i := 0; x < r.X+r.W-unit; i++ {
		w := widths[i%len(widths)] * unit
		if i%2 == 0 {
			p.Fill(ui.Rect{X: x, Y: r.Y + 1, W: w, H: r.H - 2}, ink, 0)
		}
		x += w
	}
}

// paintSketchQR 用定位图案和点阵表示二维码
func paintSketchQR(p *ui.Painter, r ui.Rect) {
	ink := ui.Hex("#111827")
	p.Fill(r, colors.Paper, 0)
	side := min(r.W, r.H)
	if side < 12 {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: side, H: side}, ui.Hex("#475467"), 0)
		return
	}
	cell := side / 21
	if cell < 1 {
		cell = 1
	}
	size := cell * 21
	x0 := r.X + (r.W-size)/2
	y0 := r.Y + (r.H-size)/2
	finder := func(cx, cy int) {
		p.Fill(ui.Rect{X: x0 + float32(cx)*cell, Y: y0 + float32(cy)*cell, W: 7 * cell, H: 7 * cell}, ink, 0)
		p.Fill(ui.Rect{X: x0 + float32(cx+1)*cell, Y: y0 + float32(cy+1)*cell, W: 5 * cell, H: 5 * cell}, colors.Panel, 0)
		p.Fill(ui.Rect{X: x0 + float32(cx+2)*cell, Y: y0 + float32(cy+2)*cell, W: 3 * cell, H: 3 * cell}, ink, 0)
	}
	finder(0, 0)
	finder(14, 0)
	finder(0, 14)
	for row := 0; row < 21; row++ {
		for col := 0; col < 21; col++ {
			inFinder := (row < 8 && col < 8) || (row < 8 && col > 12) || (row > 12 && col < 8)
			if inFinder || (row*7+col*3+row*col)%5 != 0 {
				continue
			}
			p.Fill(ui.Rect{
				X: x0 + float32(col)*cell, Y: y0 + float32(row)*cell,
				W: cell, H: cell,
			}, ink, 0)
		}
	}
}

// paintSketchTable 绘制表格的边框和内线
func paintSketchTable(p *ui.Painter, r ui.Rect) {
	line := ui.Hex("#475467")
	p.Stroke(r, line, 0, 1)
	for i := 1; i < 3; i++ {
		x := r.X + r.W*float32(i)/3
		p.Line(x, r.Y, x, r.Y+r.H, 1, line)
	}
}
