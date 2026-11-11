package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/egoist/mygo/ui"
)

const (
	// settingsDialogWidth 是设置对话框的内容宽度
	settingsDialogWidth = 880
	// settingsNavWidth 是对话框左侧导航的宽度
	settingsNavWidth = 190
	// settingsBodyHeight 是对话框导航与内容区的高度
	settingsBodyHeight = 572
	// settingsRowHeight 是偏好列表每行的高度
	settingsRowHeight = 46
	// settingsControlWidth 是偏好列表右侧控件的宽度
	settingsControlWidth = 190
)

// settingsPage 是设置对话框中的页面
type settingsPage int

const (
	// settingsGeneral 是外观与偏好设置
	settingsGeneral settingsPage = iota
	// settingsTemplates 是模板库
	settingsTemplates
	// settingsPrint 是打印设置
	settingsPrint
	// settingsAbout 是关于本应用
	settingsAbout
)

// settingsState 保存设置对话框的打开状态和筛选条件
type settingsState struct {
	open     bool
	page     settingsPage
	query    string
	category int
	template int
}

// preferences 保存跨会话保留的用户偏好
type preferences struct {
	DefaultWidth  float64 `json:"defaultWidthMm"`
	DefaultHeight float64 `json:"defaultHeightMm"`
	DefaultDPI    int     `json:"defaultDpi"`
	ShowGrid      bool    `json:"showGrid"`
	SnapGuides    bool    `json:"snapGuides"`
	// Theme 保存外观模式：light、dark 或 system
	Theme string `json:"theme"`
	// Language 保存界面语言：zh-Hans、en 或 system
	Language string `json:"language"`
}

// defaultPreferences 返回首次启动时使用的偏好
func defaultPreferences() preferences {
	return preferences{
		DefaultWidth: 50, DefaultHeight: 30, DefaultDPI: 203,
		ShowGrid: true, SnapGuides: true,
		Theme: themeNameSystem, Language: string(languageSystem),
	}
}

// prefs 保存当前生效的偏好
var prefs = defaultPreferences()

// settingsFile 返回偏好文件路径；测试会替换成临时目录
var settingsFile = func() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "GoLabel", "settings.json")
}

// loadPreferences 读取保存的偏好，文件缺失或无效时回到默认值
func loadPreferences() {
	prefs = defaultPreferences()
	path := settingsFile()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved preferences
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}
	if !within(saved.DefaultWidth, minimumSide, maximumSide) ||
		!within(saved.DefaultHeight, minimumSide, maximumSide) ||
		(saved.DefaultDPI != 203 && saved.DefaultDPI != 300 && saved.DefaultDPI != 600) {
		return
	}
	// 旧版本没有 theme 字段，或填了不认识的值时回到跟随系统
	switch saved.Theme {
	case themeNameLight, themeNameDark, themeNameSystem:
	default:
		saved.Theme = themeNameSystem
	}
	// 同样兼容没有 language 字段的配置
	switch language(saved.Language) {
	case languageZhHans, languageEnglish, languageSystem:
	default:
		saved.Language = string(languageSystem)
	}
	prefs = saved
}

// savePreferences 将偏好写入配置文件，失败时保持内存中的设置
func savePreferences() {
	path := settingsFile()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// openSettings 打开设置对话框并切换到指定页面
func (e *editor) openSettings(page settingsPage) {
	e.settings.open, e.settings.page = true, page
}

// settingsDialog 显示设置的导航与页面内容
func (e *editor) settingsDialog(c *ui.Context) {
	s := &e.settings
	if !s.open {
		return
	}
	ui.Modal(c, &s.open, func() {
		ui.Column(c.Key("dialog")).Width(settingsDialogWidth).Children(func() {
			ui.Row(c.Key("heading")).Gap(16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr("settings.title")).FontSize(19).FontWeight(650).TextColor(colors.Heading).Grow(1)
				iconAction(c.Key("close"), tr("settings.close"), iconClose, func() { s.open = false })
			})
			hairline(c.Key("rule"))
			ui.Row(c.Key("body")).Height(settingsBodyHeight).AlignItems(ui.Stretch).Children(func() {
				e.settingsNav(c.Key("nav"))
				ui.Column(c.Key("content")).Grow(1).MinWidth(0).FillHeight().
					Padding(18, 20, 18, 22).Children(func() {
					switch s.page {
					case settingsTemplates:
						e.templatePage(c)
					case settingsPrint:
						e.printSettingsPage(c)
					case settingsAbout:
						e.aboutPage(c)
					default:
						e.generalPage(c)
					}
				})
			})
		})
	})
}

// settingsNav 显示对话框左侧的页面导航
func (e *editor) settingsNav(c *ui.Context) {
	ui.Column(c).Width(settingsNavWidth).Shrink(0).FillHeight().Padding(14, 12).Gap(2).
		Background(colors.PanelAlt).BorderWidth(0, 1, 0, 0).BorderColor(colors.Hairline).
		Children(func() {
			e.navItem(c.Key("general"), tr("settings.general"), iconSliders, settingsGeneral, 0)
			e.navItem(c.Key("templates"), tr("settings.templates"), iconGrid, settingsTemplates, len(labelTemplates))
			e.navItem(c.Key("print"), tr("settings.print"), iconPrint, settingsPrint, 0)
			e.navItem(c.Key("about"), tr("settings.about"), iconInfo, settingsAbout, 0)
		})
}

// navItem 显示一个导航项，当前页为主色图标和浅底
func (e *editor) navItem(c *ui.Context, name string, icon *ui.SVG, page settingsPage, badge int) {
	selected := e.settings.page == page
	ink, label, weight := colors.Muted, colors.Body2, 500
	if selected {
		ink, label, weight = colors.Accent, colors.Text, 600
	}
	row := ui.ButtonBase(c).Height(38).FillWidth().Padding(0, 10).Gap(9).Radius(8).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(name).AccessSelected(selected).
		Cursor(ui.CursorPointer)
	switch {
	case selected:
		row.Background(colors.Selected)
	case row.Hovered():
		row.Background(colors.Hover)
	}
	row.Children(func() {
		ui.Icon(c.Key("icon"), icon).Size(16, 16).Shrink(0).TextColor(ink)
		ui.Text(c, name).FontSize(12.5).FontWeight(weight).Grow(1).TextColor(label)
		if badge > 0 {
			ui.Row(c.Key("badge")).Height(18).MinWidth(18).Padding(0, 5).Radius(9).Shrink(0).
				Background(colors.Badge).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d", badge).FontSize(10).FontWeight(600).TextColor(colors.Muted)
			})
		}
	})
	if row.Clicked() {
		e.settings.page = page
	}
}

// generalPage 显示外观主题、界面语言和偏好设置
func (e *editor) generalPage(c *ui.Context) {
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c.Key("page")).Gap(0).Children(func() {
			ui.Text(c.Key("title"), tr("settings.general")).FontSize(20).FontWeight(650).TextColor(colors.Heading)

			ui.Column(c.Key("appearance")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
				sectionHeading(c, tr("settings.appearance"))
				ui.Row(c.Key("themes")).Gap(10).Children(func() {
					e.themeCard(c.Key("light"), tr("settings.light"), themeModeLight)
					e.themeCard(c.Key("dark"), tr("settings.dark"), themeModeDark)
					e.themeCard(c.Key("auto"), tr("settings.system"), themeModeSystem)
				})
			})

			ui.Column(c.Key("preferences")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
				sectionHeading(c, tr("settings.preferences"))
				ui.Column(c.Key("rows")).Radius(12).BorderWidth(1).
					BorderColor(colors.Divider).Background(colors.PanelAlt).Children(func() {
					e.languageRow(c.Key("language"))
					e.defaultSizeRow(c.Key("size"))
					e.defaultDPIRow(c.Key("dpi"))
					e.prefSwitchRow(c.Key("grid"), tr("settings.grid"), tr("settings.gridHint"), &prefs.ShowGrid)
					e.prefSwitchRow(c.Key("snap"), tr("settings.snap"), tr("settings.snapHint"), &prefs.SnapGuides)
				})
				ui.Row(c.Key("reset")).Margin(2, 0, 4, 0).Children(func() {
					outlineAction(c.Key("button"), tr("settings.reset"), iconSliders, true, func() {
						prefs = defaultPreferences()
						resetSystemDarkCache()
						applyTheme()
						applyLanguage()
						resetFontMenu()
						savePreferences()
						// 语言回到跟随系统后，原生菜单也要交回给系统
						appLocale(prefs.Language)
						e.status = tr("settings.resetDone")
					})
				})
			})
		})
	})
}

// languageRow 显示界面语言的下拉选择，选中后立即生效并保存
func (e *editor) languageRow(c *ui.Context) {
	options := languageOptions[:]
	labels := make([]string, len(options))
	for i, option := range options {
		labels[i] = languageTitle(option)
	}
	// 配置里出现不认识的值时，下拉框显示为跟随系统
	selected := slices.Index(options, language(prefs.Language))
	if selected < 0 {
		selected = len(options) - 1
	}
	value := labels[selected]
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr("settings.language")).FontSize(12.5).FontWeight(500).Grow(1).TextColor(colors.Body)
		if ui.Combobox(c.Key("picker"), &value, labels).Width(settingsControlWidth).Height(32).Radius(7).
			FontSize(12).Label(tr("settings.languageRow")).Changed() {
			for i, label := range labels {
				if label == value {
					useLanguage(options[i])
					break
				}
			}
		}
	})
}

// themeCard 显示一张主题预览卡片，点击后立即切换并保存
func (e *editor) themeCard(c *ui.Context, name string, mode themeMode) {
	selected := parseThemeMode(prefs.Theme) == mode
	ink, border, fill := colors.Body, colors.CardBorder, colors.Card
	if selected {
		ink, border, fill = colors.Text, colors.Accent, colors.AccentSoft
	}
	// 选中的卡片用主色加粗描边，和设计稿一致
	width := float32(1)
	if selected {
		width = 1.5
	}
	card := ui.ButtonBase(c).Grow(1).Padding(10).Gap(9).Column().Radius(10).
		AlignItems(ui.Stretch).Role(ui.RoleButton).Label(name).AccessSelected(selected).
		BorderWidth(width).BorderColor(border).Background(fill)
	if card.Hovered() && !selected {
		card.BorderColor(colors.BorderStrong)
	}
	card.Children(func() {
		ui.Box(c.Key("shot")).Height(88).Radius(6).Clip().
			Draw(func(p *ui.Painter, r ui.Rect) { paintThemeShot(p, r, mode) })
		ui.Row(c.Key("caption")).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, name).FontSize(12.5).FontWeight(600).SingleLine().Grow(1).
				TextColor(ink)
			if selected {
				ui.Row(c.Key("check")).Size(16, 16).Radius(8).Shrink(0).
					Background(colors.Primary).Justify(ui.Center).AlignItems(ui.Center).
					Children(func() {
						ui.Icon(c.Key("icon"), iconCheck).Size(9, 9).TextColor(colors.OnAccent)
					})
			}
		})
	})
	if card.Clicked() && !selected {
		useThemeMode(mode)
		e.status = trf("settings.themeSwitched", themeModeTitle(mode))
	}
}

// themePalette 是一张主题缩略图使用的颜色
type themePalette struct {
	window, panel, canvas, paper, line ui.Color
}

// themePalettes 返回浅色和深色缩略图的配色
func themePalettes() (themePalette, themePalette) {
	light := themePalette{
		// 缩略图固定展示两种外观本身，不跟随当前主题
		window: ui.Hex("#FFFFFF"), panel: ui.Hex("#FBFCFD"), canvas: ui.Hex("#E8EEF5"),
		paper: ui.Hex("#FFFFFF"), line: ui.Hex("#DDE5EF"),
	}
	dark := themePalette{
		// 深色缩略图取深色画板的色值：顶栏 #1B212B、面板 #171D26、画布 #11161E
		window: ui.Hex("#1B212B"), panel: ui.Hex("#171D26"), canvas: ui.Hex("#11161E"),
		paper: ui.Hex("#E8EDF5"), line: ui.Hex("#2C3442"),
	}
	return light, dark
}

// paintMiniWindow 在缩略图区域内画一个小号编辑器窗口
func paintMiniWindow(p *ui.Painter, r ui.Rect, palette themePalette) {
	p.Fill(r, palette.panel, 0)
	top := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 12}
	p.Fill(top, palette.window, 0)
	p.Line(r.X, r.Y+12, r.X+r.W, r.Y+12, 1, palette.line)
	for i := 0; i < 3; i++ {
		p.Fill(ui.Rect{X: r.X + 5 + float32(i)*4, Y: r.Y + 5, W: 2, H: 2}, palette.line, 1)
	}
	body := ui.Rect{X: r.X, Y: r.Y + 13, W: r.W, H: r.H - 13}
	left := ui.Rect{X: body.X, Y: body.Y, W: body.W * 0.23, H: body.H}
	right := ui.Rect{X: body.X + body.W*0.77, Y: body.Y, W: body.W * 0.23, H: body.H}
	p.Fill(left, palette.panel, 0)
	p.Fill(right, palette.panel, 0)
	p.Line(left.X+left.W, body.Y, left.X+left.W, body.Y+body.H, 1, palette.line)
	p.Line(right.X, body.Y, right.X, body.Y+body.H, 1, palette.line)
	stage := ui.Rect{X: left.X + left.W + 4, Y: body.Y + 7, W: right.X - left.X - left.W - 10, H: body.H - 14}
	p.Fill(stage, palette.canvas, 0)
	paper := ui.Rect{
		X: stage.X + stage.W*0.18, Y: stage.Y + stage.H*0.2,
		W: stage.W * 0.64, H: stage.H * 0.6,
	}
	p.Fill(paper, palette.paper, 0)
	p.Stroke(paper, palette.line, 0, 1)
}

// paintThemeShot 绘制主题卡片中的界面缩略图
func paintThemeShot(p *ui.Painter, r ui.Rect, mode themeMode) {
	light, dark := themePalettes()
	switch mode {
	case themeModeLight:
		p.Fill(r, ui.Hex("#F2F5F9"), 0)
		paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, light)
	case themeModeDark:
		p.Fill(r, ui.Hex("#0E1218"), 0)
		paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, dark)
	default:
		half := r.W / 2
		p.Clip(ui.Rect{X: r.X, Y: r.Y, W: half, H: r.H}, 0, func() {
			p.Fill(r, ui.Hex("#F2F5F9"), 0)
			paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, light)
		})
		p.Clip(ui.Rect{X: r.X + half, Y: r.Y, W: r.W - half, H: r.H}, 0, func() {
			p.Fill(r, ui.Hex("#0E1218"), 0)
			paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, dark)
		})
	}
	// 缩略图外框：浅色用浅描边，深色和跟随系统用画板里的深描边
	border := ui.Hex("#E1E7EF")
	if mode != themeModeLight {
		border = ui.Hex("#2C3442")
	}
	p.Stroke(r, border, 0, 1)
}

// defaultSizeRow 显示默认标签尺寸的下拉选择
func (e *editor) defaultSizeRow(c *ui.Context) {
	value := fmt.Sprintf("%g × %g mm", prefs.DefaultWidth, prefs.DefaultHeight)
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, tr("settings.defaultSize")).FontSize(12.5).FontWeight(500).TextColor(colors.Body)
			ui.Text(c, tr("settings.defaultSizeHint")).FontSize(10.5).SingleLine().TextColor(colors.Faint)
		})
		options := make([]string, 0, len(newLabelPresets))
		for _, preset := range newLabelPresets {
			options = append(options, preset.label)
		}
		if ui.Combobox(c.Key("size"), &value, options).Width(settingsControlWidth).Height(32).Radius(7).
			FontSize(12).Label(tr("settings.defaultSize")).Changed() {
			for _, preset := range newLabelPresets {
				if preset.label == value {
					prefs.DefaultWidth, prefs.DefaultHeight = preset.width, preset.height
					savePreferences()
					break
				}
			}
		}
	})
}

// defaultDPIRow 显示默认打印分辨率的下拉选择
func (e *editor) defaultDPIRow(c *ui.Context) {
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(colors.Hairline).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, tr("settings.defaultDPI")).FontSize(12.5).FontWeight(500).TextColor(colors.Body)
			ui.Text(c, tr("settings.defaultDPIHint")).FontSize(10.5).SingleLine().TextColor(colors.Faint)
		})
		resolutions := [...]int{203, 300, 600}
		options := make([]string, len(resolutions))
		for i, dpi := range resolutions {
			options[i] = fmt.Sprintf("%d DPI", dpi)
		}
		selected := max(0, slices.Index(resolutions[:], prefs.DefaultDPI))
		value := options[selected]
		if ui.Combobox(c.Key("dpi"), &value, options).Width(settingsControlWidth).Height(32).Radius(7).
			FontSize(12).Label(tr("settings.defaultDPI")).Changed() {
			if i := slices.Index(options, value); i >= 0 {
				prefs.DefaultDPI = resolutions[i]
				savePreferences()
			}
		}
	})
}

// prefSwitchRow 显示一行带开关的偏好设置
func (e *editor) prefSwitchRow(c *ui.Context, label, hint string, value *bool) {
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(colors.Hairline).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(colors.Body)
			ui.Text(c, hint).FontSize(10.5).SingleLine().TextColor(colors.Faint)
		})
		before := *value
		ui.Switch(c.Key("switch"), value).Label(label + tr("settings.switchSuffix"))
		if before != *value {
			savePreferences()
		}
	})
}

// printSettingsPage 显示当前标签的打印参数与预览入口
func (e *editor) printSettingsPage(c *ui.Context) {
	l := e.label
	ui.Column(c).Gap(0).Children(func() {
		ui.Text(c.Key("title"), tr("settings.print")).FontSize(20).FontWeight(650).TextColor(colors.Heading)

		ui.Row(c.Key("summary")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
			printSettingCard(c.Key("paper"), iconPaper, tr("settings.paper"), fmt.Sprintf("%g × %g mm", l.Width, l.Height))
			printSettingCard(c.Key("dpi"), iconPrint, tr("inspector.resolution"), fmt.Sprintf("%d DPI", l.DPI))
			printSettingCard(c.Key("items"), iconStack, tr("settings.elements"), trf("settings.count", len(l.Items)))
		})

		ui.Column(c.Key("resolution")).Margin(22, 0, 0, 0).Gap(6).Children(func() {
			sectionHeading(c, tr("sidebar.dpi"))
			ui.Column(c.Key("picker")).Width(300).Children(func() {
				resolutions := [...]int{203, 300, 600}
				selected := max(0, slices.Index(resolutions[:], l.DPI))
				segmentedRow(c.Key("segments"), tr("sidebar.dpi"), []string{"203 DPI", "300 DPI", "600 DPI"}, selected, func(i int) {
					if l.DPI != resolutions[i] {
						l.DPI = resolutions[i]
						e.changed()
					}
				})
			})
		})

		ui.Column(c.Key("tips")).Margin(24, 0, 0, 0).Gap(7).Children(func() {
			sectionHeading(c, tr("settings.beforePrint"))
			for _, tip := range []string{
				tr("settings.tipDriver"),
				tr("settings.tipSample"),
				tr("settings.tipPDF"),
			} {
				ui.Row(c.Key(tip)).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Box(c.Key("dot")).Size(4, 4).Radius(2).Shrink(0).Margin(6, 0, 0, 0).
						Background(colors.Faint)
					ui.Text(c, tip).FontSize(11.5).Grow(1).TextColor(colors.Muted)
				})
			}
		})

		ui.Row(c.Key("actions")).Margin(24, 0, 0, 0).Gap(8).Children(func() {
			outlineAction(c.Key("preview"), tr("print.openPreview"), iconPrint, len(l.Items) > 0, func() {
				e.settings.open = false
				e.openPrintPreview()
			})
			outlineAction(c.Key("pdf"), tr("print.export"), iconDownload, len(l.Items) > 0, func() {
				e.settings.open = false
				e.perform(e.exportPDF)
			})
		})
	})
}

// printSettingCard 显示打印设置页的一个信息卡片
func printSettingCard(c *ui.Context, icon *ui.SVG, label, value string) {
	ui.Row(c).Grow(1).Height(64).Padding(0, 12).Gap(10).Radius(10).
		BorderWidth(1).BorderColor(colors.CardBorder).Background(colors.Card).
		AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("icon")).Size(30, 30).Radius(7).Shrink(0).BorderWidth(1).
			BorderColor(colors.FieldBorder).Background(colors.Panel).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c.Key("glyph"), icon).Size(15, 15).TextColor(colors.Muted)
		})
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, label).FontSize(10.5).TextColor(colors.Muted)
			ui.Text(c, value).FontSize(12.5).FontWeight(600).SingleLine().TextColor(colors.Text)
		})
	})
}

// aboutPage 显示应用版本和运行环境
func (e *editor) aboutPage(c *ui.Context) {
	ui.Column(c).Gap(0).Children(func() {
		ui.Text(c.Key("title"), tr("settings.about")).FontSize(20).FontWeight(650).TextColor(colors.Heading)

		ui.Row(c.Key("brand")).Margin(20, 0, 0, 0).Gap(14).AlignItems(ui.Center).Children(func() {
			ui.Row(c.Key("mark")).Size(48, 48).Radius(12).Shrink(0).Background(colors.Primary).
				Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Icon(c.Key("icon"), iconLogo).Size(26, 26).TextColor(colors.Panel)
			})
			ui.Column(c.Key("text")).Gap(3).Children(func() {
				ui.Text(c, "GoLabel").FontSize(18).FontWeight(650).TextColor(colors.Heading)
				ui.Text(c, tr("settings.tagline")).FontSize(11.5).TextColor(colors.Muted)
			})
		})

		ui.Column(c.Key("rows")).Margin(24, 0, 0, 0).Radius(12).BorderWidth(1).
			BorderColor(colors.Divider).Background(colors.PanelAlt).Children(func() {
			aboutRow(c.Key("version"), tr("settings.version"), "0.1.0")
			aboutRow(c.Key("runtime"), tr("settings.runtime"), runtime.Version()+" · "+runtime.GOOS+"/"+runtime.GOARCH)
			aboutRow(c.Key("ui"), tr("settings.framework"), "MyGo Native UI 0.8.3")
			aboutRow(c.Key("format"), tr("settings.format"), tr("settings.formatValue"))
			aboutRow(c.Key("license"), tr("settings.license"), "MIT License")
		})

		ui.Text(c, tr("settings.privacy")).
			FontSize(11).Margin(18, 0, 0, 0).TextColor(colors.Muted)
	})
}

// aboutRow 显示关于页的一行信息
func aboutRow(c *ui.Context, label, value string) {
	ui.Row(c).Height(42).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).Shrink(0).Width(100).TextColor(colors.Muted)
		ui.Text(c, value).FontSize(12).SingleLine().Grow(1).TextColor(colors.Text)
	})
}
