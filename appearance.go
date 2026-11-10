package main

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

// 界面配色取自 designs/golabel-redesign，集中定义便于统一调整
const (
	// colorCanvas 是画布区域的底色
	colorCanvas = "#EDF0F4"
	// colorPanel 是两侧面板和顶栏的底色
	colorPanel = "#FFFFFF"
	// colorDivider 是面板与顶栏之间的分隔线
	colorDivider = "#E6EAF0"
	// colorHairline 是面板内部分组之间的细线
	colorHairline = "#EDF0F4"
	// colorFieldBorder 是输入框和卡片的描边
	colorFieldBorder = "#E1E7EF"
	// colorFieldBg 是输入框和工具卡片的浅底色
	colorFieldBg = "#FBFCFE"
	// colorChip 是图标底色块
	colorChip = "#F4F6FA"
	// colorLayerChip 是图层图标底色块
	colorLayerChip = "#F8FAFC"
	// colorHeading 是标题文字
	colorHeading = "#101828"
	// colorText 是正文文字
	colorText = "#1F2A44"
	// colorBody 是工具栏按钮文字
	colorBody = "#344054"
	// colorMuted 是分组标题和次要文字
	colorMuted = "#667085"
	// colorFaint 是单位、尺寸等辅助文字
	colorFaint = "#98A2B3"
	// colorAccent 是主色
	colorAccent = "#2F52E0"
	// colorDot 是画布点阵
	colorDot = "#D8E0EA"
	// colorRuler 是画布标注线
	colorRuler = "#C9D3DF"
	// colorRulerText 是画布标注文字
	colorRulerText = "#8A94A6"
	// colorSuccess 和 colorSuccessSoft 是已保存提示
	colorSuccess     = "#067647"
	colorSuccessSoft = "#ECFDF3"
	// colorWarning 和 colorWarningSoft 是未保存提示
	colorWarning     = "#B54708"
	colorWarningSoft = "#FFFAEB"
	// colorGuide 是拖拽时的对齐线
	colorGuide = "#E33678"
	// colorHover 是次要按钮的悬停底色
	colorHover = "#F2F4F7"
	// colorSelected 是图层选中行底色
	colorSelected = "#EFF3F8"
	// colorLayerHover 是图层悬停行底色
	colorLayerHover = "#F6F8FB"
	// colorPaperBorder 是纸张描边
	colorPaperBorder = "#DFE5EC"
)

// interfaceTheme 返回界面统一使用的浅色主题
func interfaceTheme() *ui.Theme {
	t := ui.LightTheme()
	// 主题底色留白，画布区域的灰色由编辑器自己铺，弹层和对话框因此保持白色
	t.Background = ui.Hex(colorPanel)
	t.Surface = ui.Hex(colorChip)
	t.SurfaceHover = ui.Hex("#EDF1F6")
	t.SurfacePressed = ui.Hex("#E4E9F2")
	t.Border = ui.Hex(colorFieldBorder)
	t.Text = ui.Hex(colorText)
	t.TextMuted = ui.Hex(colorMuted)
	t.Accent = ui.Hex(colorAccent)
	t.AccentHover = ui.Hex("#2743C4")
	t.AccentPressed = ui.Hex("#1E36A8")
	t.AccentText = ui.Hex(colorPanel)
	t.InactiveAccent = ui.Hex(colorAccent)
	t.Danger = ui.Hex("#D92D20")
	t.Warning = ui.Hex("#DC6803")
	t.Success = ui.Hex(colorSuccess)
	t.Selection = ui.RGBA(47, 82, 224, 0.22)
	t.Focus = ui.RGBA(47, 82, 224, 0.45)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1
	t.Scrollbar = ui.RGBA(16, 24, 40, 0.18)
	t.ScrollbarWidth = 8
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	return t
}

// appTheme 只构建一次界面主题
var appTheme = sync.OnceValue(interfaceTheme)

// svgIcon 用统一的线宽和圆角端点构建线性图标
func svgIcon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" ` +
		`stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

// 界面图标，路径取自设计稿
var (
	iconLogo     = svgIcon(`<path d="M4 7.5A2.5 2.5 0 0 1 6.5 5h11A2.5 2.5 0 0 1 20 7.5v9a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 16.5v-9Z"/><path d="M8 9.5h8M8 13h5"/>`)
	iconPlus     = svgIcon(`<path d="M12 5v14M5 12h14"/>`)
	iconFolder   = svgIcon(`<path d="M3.5 7.5A1.5 1.5 0 0 1 5 6h4l2 2h8a1.5 1.5 0 0 1 1.5 1.5v8A1.5 1.5 0 0 1 19 19H5a1.5 1.5 0 0 1-1.5-1.5v-10Z"/>`)
	iconSave     = svgIcon(`<path d="M5 5h11l3 3v11H5V5Z"/><path d="M8 5v5h7V5M8 19v-6h8v6"/>`)
	iconDownload = svgIcon(`<path d="M12 4v11m-4-4 4 4 4-4M5 20h14"/>`)
	iconPrint    = svgIcon(`<path d="M7 9V4h10v5M7 17H5a1.5 1.5 0 0 1-1.5-1.5v-5A1.5 1.5 0 0 1 5 9h14a1.5 1.5 0 0 1 1.5 1.5v5A1.5 1.5 0 0 1 19 17h-2"/><path d="M7 14h10v7H7v-7Z"/>`)
	iconText     = svgIcon(`<path d="M5 5h14M12 5v14M9 19h6"/>`)
	iconPicture  = svgIcon(`<rect x="4" y="5" width="16" height="14" rx="1.5"/><path d="m4 16 4.5-4.5L13 16l3-3 4 4"/><circle cx="15.5" cy="9" r="1.4" fill="currentColor" stroke="none"/>`)
	iconBarcode  = svgIcon(`<path d="M4 5v14M7.5 5v14M11 5v14M14.5 5v14M18 5v14M21 5v14"/>`)
	iconQR       = svgIcon(`<rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><path d="M14 14h2v2h-2zM18 14h2v2h-2zM14 18h2v2h-2zM18 18h2v2h-2z" fill="currentColor" stroke="none"/>`)
	iconLine     = svgIcon(`<path d="M4 12h16"/><circle cx="4" cy="12" r="1.3" fill="currentColor" stroke="none"/><circle cx="20" cy="12" r="1.3" fill="currentColor" stroke="none"/>`)
	iconTable    = svgIcon(`<rect x="4" y="4" width="16" height="16" rx="1.5"/><path d="M4 9.3h16M4 14.6h16M9.3 4v16M14.6 4v16"/>`)
	iconStack    = svgIcon(`<path d="m12 3 8 4.5-8 4.5-8-4.5L12 3Z"/><path d="m4 12 8 4.5 8-4.5M4 16.5 12 21l8-4.5"/>`)
	iconChevron  = svgIcon(`<path d="m9 6 6 6-6 6"/>`)
	iconMinus    = svgIcon(`<path d="M5 12h14"/>`)
	iconFit      = svgIcon(`<path d="M4 9V5h4M20 9V5h-4M4 15v4h4M20 15v4h-4"/>`)
	iconTrash    = svgIcon(`<path d="M5 7h14m-9-3h4M7 7l1 13h8l1-13m-6 3v6m2-6v6"/>`)
	iconPaper    = svgIcon(`<rect x="4" y="5" width="16" height="14" rx="1.5"/><path d="M8 9h8M8 12h5"/>`)
	iconSliders  = svgIcon(`<path d="M4 7h9M17 7h3M4 17h3M11 17h9"/><circle cx="15" cy="7" r="2.4"/><circle cx="8" cy="17" r="2.4"/>`)
	iconGrid     = svgIcon(`<rect x="4" y="4" width="7" height="7" rx="1.5"/><rect x="13" y="4" width="7" height="7" rx="1.5"/><rect x="4" y="13" width="7" height="7" rx="1.5"/><rect x="13" y="13" width="7" height="7" rx="1.5"/>`)
	iconInfo     = svgIcon(`<circle cx="12" cy="12" r="8.5"/><path d="M12 11v5M12 8h.01"/>`)
	iconSearch   = svgIcon(`<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>`)
	iconClose    = svgIcon(`<path d="m6 6 12 12M18 6 6 18"/>`)
	iconCheck    = svgIcon(`<path d="m5 12 5 5 9-10"/>`)
)

// layerIcon 返回元素在图层列表中的图标
func layerIcon(kind itemKind) *ui.SVG {
	switch kind {
	case kindImage:
		return iconPicture
	case kindBarcode:
		return iconBarcode
	case kindQR:
		return iconQR
	case kindLine:
		return iconLine
	case kindTable:
		return iconTable
	default:
		return iconText
	}
}

// kindTitle 返回元素在属性栏顶部的名称
func kindTitle(kind itemKind) string {
	switch kind {
	case kindImage:
		return "图片"
	case kindBarcode:
		return "条码"
	case kindQR:
		return "二维码"
	case kindLine:
		return "分隔线"
	case kindTable:
		return "表格"
	default:
		return "文字"
	}
}

// sectionHeading 显示属性栏和侧栏的小号分组标题
func sectionHeading(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(11).FontWeight(650).LetterSpacing(0.5).TextColor(ui.Hex(colorMuted))
}

// hairline 显示面板内的分组分隔线
func hairline(c *ui.Context) {
	ui.Box(c).Height(1).FillWidth().Shrink(0).Margin(8, 0).Background(ui.Hex(colorHairline))
}

// secondaryButton 显示描边的次要按钮，返回是否被点击
func secondaryButton(c *ui.Context, label string, enabled bool) bool {
	ink, border, fill := colorBody, colorFieldBorder, colorPanel
	if !enabled {
		ink, border, fill = "#B3BCCB", colorDivider, colorFieldBg
	}
	button := ui.ButtonBase(c).Height(34).FillWidth().Radius(8).Role(ui.RoleButton).
		Label(label).Disabled(!enabled).Justify(ui.Center).AlignItems(ui.Center).
		BorderWidth(1).BorderColor(ui.Hex(border)).Background(ui.Hex(fill))
	if button.Hovered() && enabled {
		button.Background(ui.Hex(colorFieldBg)).BorderColor(ui.Hex("#C9D3DF"))
	}
	button.Children(func() {
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(ui.Hex(ink))
	})
	return button.Clicked() && enabled
}

// optionRow 显示一排等宽的分段选项，选中项为白底加粗
func optionRow(c *ui.Context, labels []string, selected int, choose func(int)) {
	segmentedRow(c, "", labels, selected, choose)
}

// segmentedRow 与 optionRow 相同，name 会加在无障碍名称前以便区分同名选项
func segmentedRow(c *ui.Context, name string, labels []string, selected int, choose func(int)) {
	ui.Row(c).Padding(2).Gap(2).Radius(7).Background(ui.Hex(colorChip)).Children(func() {
		for i, label := range labels {
			accessible := label
			if name != "" {
				accessible = name + " " + label
			}
			button := ui.ButtonBase(c.Key(label)).Grow(1).Height(28).Radius(5).
				Role(ui.RoleButton).Label(accessible).AccessSelected(i == selected)
			ink, weight := colorMuted, 500
			if i == selected {
				button.Background(ui.Hex(colorPanel)).Shadow(0, 1, 2, 0, ui.RGBA(16, 24, 40, 0.08))
				ink, weight = colorText, 600
			}
			button.Children(func() {
				ui.Text(c, label).FontSize(11).FontWeight(weight).TextColor(ui.Hex(ink))
			})
			if button.Clicked() && i != selected {
				choose(i)
			}
		}
	})
}
