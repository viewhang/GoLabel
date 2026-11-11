package main

import (
	"github.com/egoist/mygo/ui"
)

// palette 是一套完整的界面配色。浅色和深色各有一份，切换主题时整体替换，
// 界面里的颜色都从 colors 取值，因此换主题不需要重建任何组件。
type palette struct {
	// Dark 表示这是一套深色配色
	Dark bool

	// 底色：画布、顶栏、面板、成组卡片和各类按钮
	Canvas, Header, Panel, PanelAlt, FieldBg, Stage ui.Color
	// Card 是浮在面板上的卡片底色，Tile 是面板里的工具卡，
	// Floating 是画布上的浮层；TileHover 是工具卡悬停时的底色
	Card, Tile, TileHover, Floating                  ui.Color
	Chip, ChipAlt, ChipPressed, Segmented, LayerChip ui.Color
	// ChipSelected 是面板上单个按钮的选中底色（分段控件用 Segmented）
	ChipSelected                                   ui.Color
	Hover, Selected, LayerHover, Badge, AccentSoft ui.Color

	// 描边与纸张
	Divider, Hairline, FieldBorder, CardBorder ui.Color
	BorderHover, BorderStrong, DashBorder      ui.Color
	Paper, PaperBorder                         ui.Color
	// 画在白纸上的浮层固定使用浅色，纸张在深浅主题下都是白的
	PaperInk, PaperMuted, PaperLine, PaperChip ui.Color

	// 文字；Section 用于「外观」「添加元素」这类分组小标题
	Heading, Text, Body, Body2, Muted, Faint, Disabled, Section ui.Color

	// 主色与状态：Accent 用于文字和描边，Primary 用于实心主按钮
	Accent, AccentHover, AccentPressed, OnAccent       ui.Color
	Primary, PrimaryHover, PrimaryPressed              ui.Color
	Success, SuccessSoft, Warning, WarningSoft, Danger ui.Color

	// 画布上的点阵、标尺和对齐线
	Dot, Ruler, RulerText, Guide, SketchDot ui.Color

	// 阴影，按深浅分档
	ShadowSoft, ShadowSubtle, ShadowMid, ShadowStrong, ShadowDeep ui.Color
}

// lightPalette 是浅色主题，配色取自 designs/golabel-redesign
var lightPalette = palette{
	Canvas:    ui.Hex("#EDF0F4"),
	Header:    ui.Hex("#FFFFFF"),
	Panel:     ui.Hex("#FFFFFF"),
	PanelAlt:  ui.Hex("#FBFCFD"),
	FieldBg:   ui.Hex("#FBFCFE"),
	Stage:     ui.Hex("#F4F6F9"),
	Card:      ui.Hex("#FFFFFF"),
	Tile:      ui.Hex("#FBFCFE"),
	TileHover: ui.Hex("#FFFFFF"),
	Floating:  ui.Hex("#FFFFFF"),

	Chip:         ui.Hex("#F4F6FA"),
	ChipAlt:      ui.Hex("#F2F5F9"),
	ChipPressed:  ui.Hex("#E9EEF5"),
	Segmented:    ui.Hex("#FFFFFF"),
	ChipSelected: ui.Hex("#F4F6FA"),
	LayerChip:    ui.Hex("#F8FAFC"),
	Hover:        ui.Hex("#F2F4F7"),
	Selected:     ui.Hex("#EFF3F8"),
	LayerHover:   ui.Hex("#F6F8FB"),
	Badge:        ui.Hex("#E4E9F0"),
	AccentSoft:   ui.Hex("#F8FAFF"),

	Divider:      ui.Hex("#E6EAF0"),
	Hairline:     ui.Hex("#EDF0F4"),
	FieldBorder:  ui.Hex("#E1E7EF"),
	CardBorder:   ui.Hex("#E8ECF2"),
	BorderHover:  ui.Hex("#C9D3DF"),
	BorderStrong: ui.Hex("#C7D2E3"),
	DashBorder:   ui.Hex("#DDE5EF"),
	Paper:        ui.Hex("#FFFFFF"),
	PaperBorder:  ui.Hex("#DFE5EC"),

	Heading:  ui.Hex("#101828"),
	Text:     ui.Hex("#1F2A44"),
	Body:     ui.Hex("#344054"),
	Body2:    ui.Hex("#475467"),
	Muted:    ui.Hex("#667085"),
	Faint:    ui.Hex("#98A2B3"),
	Disabled: ui.Hex("#B3BCCB"),
	Section:  ui.Hex("#667085"),

	Accent:        ui.Hex("#2F52E0"),
	AccentHover:   ui.Hex("#2743C4"),
	AccentPressed: ui.Hex("#1E36A8"),
	OnAccent:      ui.Hex("#FFFFFF"),

	Primary:        ui.Hex("#2F52E0"),
	PrimaryHover:   ui.Hex("#2743C4"),
	PrimaryPressed: ui.Hex("#1E36A8"),

	Success:     ui.Hex("#067647"),
	SuccessSoft: ui.Hex("#ECFDF3"),
	Warning:     ui.Hex("#B54708"),
	WarningSoft: ui.Hex("#FFFAEB"),
	Danger:      ui.Hex("#D92D20"),

	Dot:       ui.Hex("#D8E0EA"),
	Ruler:     ui.Hex("#C9D3DF"),
	RulerText: ui.Hex("#8A94A6"),
	Guide:     ui.Hex("#E33678"),
	SketchDot: ui.Hex("#DCE3EB"),

	ShadowSoft:   ui.RGBA(16, 24, 40, 0.08),
	ShadowSubtle: ui.RGBA(16, 24, 40, 0.10),
	ShadowMid:    ui.RGBA(16, 24, 40, 0.16),
	ShadowStrong: ui.RGBA(16, 24, 40, 0.22),
	ShadowDeep:   ui.RGBA(16, 24, 40, 0.26),
}

// darkPalette 是深色主题，配色取自 designs/golabel-redesign 的深色画板：
// 画布 #0E1218，面板 #151A22，成组控件 #11161E，主色文字 #7C9AFF，
// 实心主按钮 #3D5AFE；标签纸仍是白色，不随主题变暗
var darkPalette = palette{
	Dark: true,

	Canvas:    ui.Hex("#0E1218"),
	Header:    ui.Hex("#12171F"),
	Panel:     ui.Hex("#151A22"),
	PanelAlt:  ui.Hex("#11161E"),
	FieldBg:   ui.Hex("#11161E"),
	Stage:     ui.Hex("#11161E"),
	Card:      ui.Hex("#171D26"),
	Tile:      ui.Hex("#171D26"),
	TileHover: ui.Hex("#1D2430"),
	Floating:  ui.Hex("#1B212B"),

	Chip:         ui.Hex("#11161E"),
	ChipAlt:      ui.Hex("#171D26"),
	ChipPressed:  ui.Hex("#1D2430"),
	Segmented:    ui.Hex("#232C3C"),
	ChipSelected: ui.Hex("#232C3C"),
	LayerChip:    ui.Hex("#171D26"),
	Hover:        ui.Hex("#1D2430"),
	Selected:     ui.Hex("#1E2632"),
	LayerHover:   ui.Hex("#1A202A"),
	Badge:        ui.Hex("#202834"),
	AccentSoft:   ui.Hex("#1A2130"),

	Divider:      ui.Hex("#232B37"),
	Hairline:     ui.Hex("#232B37"),
	FieldBorder:  ui.Hex("#2C3442"),
	CardBorder:   ui.Hex("#262E3B"),
	BorderHover:  ui.Hex("#3A4454"),
	BorderStrong: ui.Hex("#354052"),
	DashBorder:   ui.Hex("#2C3442"),
	// 纸张始终是白色的标签，深色下也不改，只让描边收进背景
	Paper:       ui.Hex("#FFFFFF"),
	PaperBorder: ui.Hex("#3A4454"),

	// 画在白纸上的浮层固定使用浅色，纸张在深浅主题下都是白的
	PaperInk:   ui.Hex("#111827"),
	PaperMuted: ui.Hex("#6B7280"),
	PaperLine:  ui.Hex("#DFE5EC"),
	PaperChip:  ui.Hex("#F2F5F9"),

	Heading:  ui.Hex("#F2F5FA"),
	Text:     ui.Hex("#E8EDF5"),
	Body:     ui.Hex("#D5DCE8"),
	Body2:    ui.Hex("#C3CCDA"),
	Muted:    ui.Hex("#A7B2C3"),
	Faint:    ui.Hex("#7E8A9D"),
	Disabled: ui.Hex("#6F7B8D"),
	Section:  ui.Hex("#7E8A9D"),

	Accent:        ui.Hex("#7C9AFF"),
	AccentHover:   ui.Hex("#94ADFF"),
	AccentPressed: ui.Hex("#6485F0"),
	OnAccent:      ui.Hex("#FFFFFF"),

	Primary:        ui.Hex("#3D5AFE"),
	PrimaryHover:   ui.Hex("#5470FF"),
	PrimaryPressed: ui.Hex("#2F49D8"),

	Success:     ui.Hex("#6EE7A8"),
	SuccessSoft: ui.Hex("#10281D"),
	Warning:     ui.Hex("#F5B86E"),
	WarningSoft: ui.Hex("#2E2416"),
	Danger:      ui.Hex("#FF6B63"),

	Dot:       ui.Hex("#232B37"),
	Ruler:     ui.Hex("#2C3442"),
	RulerText: ui.Hex("#8A96A8"),
	Guide:     ui.Hex("#FF5C9E"),
	SketchDot: ui.Hex("#232B37"),

	ShadowSoft:   ui.RGBA(0, 0, 0, 0.30),
	ShadowSubtle: ui.RGBA(0, 0, 0, 0.35),
	ShadowMid:    ui.RGBA(0, 0, 0, 0.45),
	ShadowStrong: ui.RGBA(0, 0, 0, 0.55),
	ShadowDeep:   ui.RGBA(0, 0, 0, 0.60),
}

// colors 是当前生效的配色，由 applyTheme 按偏好整体替换
var colors = lightPalette

// buildTheme 按一套配色构建界面主题
func buildTheme(p palette) *ui.Theme {
	t := ui.LightTheme()
	if p.Dark {
		t = ui.DarkTheme()
	}
	// 主题底色留白，画布区域的灰色由编辑器自己铺，弹层和对话框因此保持面板色
	t.Background = p.Panel
	t.Surface = p.Chip
	t.SurfaceHover = p.Hover
	t.SurfacePressed = p.ChipPressed
	t.Border = p.FieldBorder
	t.Text = p.Text
	t.TextMuted = p.Muted
	// 框架自己的控件（开关、勾选、主按钮）用实心主色，界面里的文字和描边用浅一号的 Accent
	t.Accent = p.Primary
	t.AccentHover = p.PrimaryHover
	t.AccentPressed = p.PrimaryPressed
	t.AccentText = p.OnAccent
	t.InactiveAccent = p.Primary
	t.Danger = p.Danger
	t.Warning = p.Warning
	t.Success = p.Success
	t.FocusRingWidth = 2
	t.FocusRingOffset = 1
	t.ScrollbarWidth = 8
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13
	if p.Dark {
		t.Selection = ui.RGBA(79, 111, 240, 0.34)
		t.Focus = ui.RGBA(109, 136, 247, 0.6)
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.22)
	} else {
		t.Selection = ui.RGBA(47, 82, 224, 0.22)
		t.Focus = ui.RGBA(47, 82, 224, 0.45)
		t.Scrollbar = ui.RGBA(16, 24, 40, 0.18)
	}
	return t
}

// 浅色和深色主题各构建一次，切换时直接取用
var (
	lightAppTheme = buildTheme(lightPalette)
	darkAppTheme  = buildTheme(darkPalette)
)

// appTheme 返回当前配色对应的界面主题
func appTheme() *ui.Theme {
	if colors.Dark {
		return darkAppTheme
	}
	return lightAppTheme
}

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
	iconSun      = svgIcon(`<circle cx="12" cy="12" r="4"/><path d="M12 3v2M12 19v2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M3 12h2M19 12h2M5.6 18.4 7 17M17 7l1.4-1.4"/>`)
	iconMoon     = svgIcon(`<path d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z"/>`)
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
		return tr("kind.image")
	case kindBarcode:
		return tr("kind.barcode")
	case kindQR:
		return tr("kind.qr")
	case kindLine:
		return tr("kind.line")
	case kindTable:
		return tr("kind.table")
	default:
		return tr("kind.text")
	}
}

// sectionHeading 显示属性栏和侧栏的小号分组标题
func sectionHeading(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(11).FontWeight(650).LetterSpacing(0.5).TextColor(colors.Section)
}

// hairline 显示面板内的分组分隔线
func hairline(c *ui.Context) {
	ui.Box(c).Height(1).FillWidth().Shrink(0).Margin(8, 0).Background(colors.Hairline)
}

// secondaryButton 显示描边的次要按钮，返回是否被点击
func secondaryButton(c *ui.Context, label string, enabled bool) bool {
	ink, border, fill := colors.Body, colors.FieldBorder, colors.Panel
	if !enabled {
		ink, border, fill = colors.Disabled, colors.Divider, colors.FieldBg
	}
	button := ui.ButtonBase(c).Height(34).FillWidth().Radius(8).Role(ui.RoleButton).
		Label(label).Disabled(!enabled).Justify(ui.Center).AlignItems(ui.Center).
		BorderWidth(1).BorderColor(border).Background(fill)
	if button.Hovered() && enabled {
		button.Background(colors.FieldBg).BorderColor(colors.BorderHover)
	}
	button.Children(func() {
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(ink)
	})
	return button.Clicked() && enabled
}

// optionRow 显示一排等宽的分段选项，选中项为底加重加粗
func optionRow(c *ui.Context, labels []string, selected int, choose func(int)) {
	segmentedRow(c, "", labels, selected, choose)
}

// segmentedRow 与 optionRow 相同，name 会加在无障碍名称前以便区分同名选项
func segmentedRow(c *ui.Context, name string, labels []string, selected int, choose func(int)) {
	ui.Row(c).Padding(2).Gap(2).Radius(7).Background(colors.Chip).Children(func() {
		for i, label := range labels {
			accessible := label
			if name != "" {
				accessible = name + " " + label
			}
			button := ui.ButtonBase(c.Key(label)).Grow(1).Height(28).Radius(5).
				Role(ui.RoleButton).Label(accessible).AccessSelected(i == selected)
			ink, weight := colors.Muted, 500
			if i == selected {
				button.Background(colors.Segmented).Shadow(0, 1, 2, 0, colors.ShadowSoft)
				ink, weight = colors.Text, 600
			}
			button.Children(func() {
				ui.Text(c, label).FontSize(11).FontWeight(weight).TextColor(ink)
			})
			if button.Clicked() && i != selected {
				choose(i)
			}
		}
	})
}
