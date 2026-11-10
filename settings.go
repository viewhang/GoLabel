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
	settingsBodyHeight = 520
	// settingsRowHeight 是偏好列表每行的高度
	settingsRowHeight = 46
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
}

// defaultPreferences 返回首次启动时使用的偏好
func defaultPreferences() preferences {
	return preferences{DefaultWidth: 50, DefaultHeight: 30, DefaultDPI: 203, ShowGrid: true, SnapGuides: true}
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
				ui.Text(c, "设置").FontSize(19).FontWeight(650).TextColor(ui.Hex(colorHeading)).Grow(1)
				iconAction(c.Key("close"), "关闭", iconClose, func() { s.open = false })
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
		Background(ui.Hex("#FBFCFD")).BorderWidth(0, 1, 0, 0).BorderColor(ui.Hex(colorHairline)).
		Children(func() {
			e.navItem(c.Key("general"), "通用", iconSliders, settingsGeneral, 0)
			e.navItem(c.Key("templates"), "模板库", iconGrid, settingsTemplates, len(labelTemplates))
			e.navItem(c.Key("print"), "打印", iconPrint, settingsPrint, 0)
			e.navItem(c.Key("about"), "关于", iconInfo, settingsAbout, 0)
		})
}

// navItem 显示一个导航项，当前页为主色图标和浅底
func (e *editor) navItem(c *ui.Context, name string, icon *ui.SVG, page settingsPage, badge int) {
	selected := e.settings.page == page
	ink, label, weight := colorMuted, "#475467", 500
	if selected {
		ink, label, weight = colorAccent, colorText, 600
	}
	row := ui.ButtonBase(c).Height(38).FillWidth().Padding(0, 10).Gap(9).Radius(8).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(name).AccessSelected(selected).
		Cursor(ui.CursorPointer)
	switch {
	case selected:
		row.Background(ui.Hex(colorSelected))
	case row.Hovered():
		row.Background(ui.Hex(colorHover))
	}
	row.Children(func() {
		ui.Icon(c.Key("icon"), icon).Size(16, 16).Shrink(0).TextColor(ui.Hex(ink))
		ui.Text(c, name).FontSize(12.5).FontWeight(weight).Grow(1).TextColor(ui.Hex(label))
		if badge > 0 {
			ui.Row(c.Key("badge")).Height(18).MinWidth(18).Padding(0, 5).Radius(9).Shrink(0).
				Background(ui.Hex("#E4E9F0")).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d", badge).FontSize(10).FontWeight(600).TextColor(ui.Hex(colorMuted))
			})
		}
	})
	if row.Clicked() {
		e.settings.page = page
	}
}

// generalPage 显示外观主题和偏好设置
func (e *editor) generalPage(c *ui.Context) {
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c.Key("page")).Gap(0).Children(func() {
			ui.Text(c.Key("title"), "通用").FontSize(20).FontWeight(650).TextColor(ui.Hex(colorHeading))

			ui.Column(c.Key("appearance")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
				sectionHeading(c, "外观")
				ui.Row(c.Key("themes")).Gap(10).Children(func() {
					e.themeCard(c.Key("light"), "浅色", themeLight)
					e.themeCard(c.Key("dark"), "深色", themeDark)
					e.themeCard(c.Key("auto"), "跟随系统", themeAuto)
				})
			})

			ui.Column(c.Key("preferences")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
				sectionHeading(c, "偏好")
				ui.Column(c.Key("rows")).Radius(12).BorderWidth(1).
					BorderColor(ui.Hex(colorDivider)).Background(ui.Hex("#FBFCFD")).Children(func() {
					e.defaultSizeRow(c.Key("size"))
					e.defaultDPIRow(c.Key("dpi"))
					e.prefSwitchRow(c.Key("grid"), "显示网格", "在画布上显示点阵参考线", &prefs.ShowGrid)
					e.prefSwitchRow(c.Key("snap"), "对齐吸附", "拖动时吸附到其他元素和纸张边缘", &prefs.SnapGuides)
				})
				ui.Row(c.Key("reset")).Margin(2, 0, 4, 0).Children(func() {
					outlineAction(c.Key("button"), "恢复默认设置", iconSliders, true, func() {
						prefs = defaultPreferences()
						savePreferences()
						e.status = "已恢复默认设置"
					})
				})
			})
		})
	})
}

// themeKind 区分外观设置中的三张主题卡片
type themeKind int

const (
	themeLight themeKind = iota
	themeDark
	themeAuto
)

// themeCard 显示一张主题预览卡片，未支持的主题会被禁用
func (e *editor) themeCard(c *ui.Context, name string, kind themeKind) {
	available := kind == themeLight
	selected := available
	ink, border, fill := colorBody, colorFieldBorder, colorPanel
	if selected {
		ink, border, fill = colorText, colorAccent, "#F8FAFF"
	}
	card := ui.ButtonBase(c).Grow(1).Padding(10).Gap(9).Column().Radius(10).
		AlignItems(ui.Stretch).Role(ui.RoleButton).Label(name).AccessSelected(selected).
		Disabled(!available).BorderWidth(1).BorderColor(ui.Hex(border)).
		Background(ui.Hex(fill))
	if card.Hovered() && !selected {
		card.BorderColor(ui.Hex("#C7D2E3"))
	}
	card.Children(func() {
		ui.Box(c.Key("shot")).Height(88).Radius(6).Clip().
			Draw(func(p *ui.Painter, r ui.Rect) { paintThemeShot(p, r, kind) })
		ui.Row(c.Key("caption")).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, name).FontSize(12.5).FontWeight(600).SingleLine().Grow(1).
				TextColor(ui.Hex(ink))
			if available {
				ui.Row(c.Key("check")).Size(16, 16).Radius(8).Shrink(0).
					Background(ui.Hex(colorAccent)).Justify(ui.Center).AlignItems(ui.Center).
					Children(func() {
						ui.Icon(c.Key("icon"), iconCheck).Size(9, 9).TextColor(ui.Hex(colorPanel))
					})
			} else {
				ui.Text(c, "即将支持").FontSize(9.5).Shrink(0).TextColor(ui.Hex(colorFaint))
			}
		})
	})
	if card.Clicked() && available {
		e.status = "已使用浅色主题"
	}
}

// themePalette 是一张主题缩略图使用的颜色
type themePalette struct {
	window, panel, canvas, paper, line ui.Color
}

// themePalettes 返回浅色和深色缩略图的配色
func themePalettes() (themePalette, themePalette) {
	light := themePalette{
		window: ui.Hex(colorPanel), panel: ui.Hex("#FBFCFD"), canvas: ui.Hex("#E8EEF5"),
		paper: ui.Hex(colorPanel), line: ui.Hex("#DDE5EF"),
	}
	dark := themePalette{
		window: ui.Hex("#232A35"), panel: ui.Hex("#202732"), canvas: ui.Hex("#1E2530"),
		paper: ui.Hex("#E8EDF5"), line: ui.Hex("#323B4A"),
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
func paintThemeShot(p *ui.Painter, r ui.Rect, kind themeKind) {
	light, dark := themePalettes()
	switch kind {
	case themeLight:
		p.Fill(r, ui.Hex("#F2F5F9"), 0)
		paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, light)
	case themeDark:
		p.Fill(r, ui.Hex("#1B212B"), 0)
		paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, dark)
	default:
		half := r.W / 2
		p.Clip(ui.Rect{X: r.X, Y: r.Y, W: half, H: r.H}, 0, func() {
			p.Fill(r, ui.Hex("#F2F5F9"), 0)
			paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, light)
		})
		p.Clip(ui.Rect{X: r.X + half, Y: r.Y, W: r.W - half, H: r.H}, 0, func() {
			p.Fill(r, ui.Hex("#1B212B"), 0)
			paintMiniWindow(p, ui.Rect{X: r.X + 6, Y: r.Y + 6, W: r.W - 12, H: r.H - 12}, dark)
		})
	}
	p.Stroke(r, ui.Hex("#D5DEE8"), 0, 1)
}

// defaultSizeRow 显示默认标签尺寸的下拉选择
func (e *editor) defaultSizeRow(c *ui.Context) {
	value := fmt.Sprintf("%g × %g mm", prefs.DefaultWidth, prefs.DefaultHeight)
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, "默认尺寸").FontSize(12.5).FontWeight(500).TextColor(ui.Hex(colorBody))
			ui.Text(c, "新建标签时预先选中的规格").FontSize(10.5).SingleLine().TextColor(ui.Hex(colorFaint))
		})
		options := make([]string, 0, len(newLabelPresets))
		for _, preset := range newLabelPresets {
			options = append(options, preset.label)
		}
		if ui.Combobox(c.Key("size"), &value, options).Width(178).Height(32).Radius(7).
			FontSize(12).Label("默认尺寸").Changed() {
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

// defaultDPIRow 显示默认打印分辨率的分段选择
func (e *editor) defaultDPIRow(c *ui.Context) {
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(ui.Hex(colorHairline)).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, "默认分辨率").FontSize(12.5).FontWeight(500).TextColor(ui.Hex(colorBody))
			ui.Text(c, "标签打印机的打印头 DPI").FontSize(10.5).SingleLine().TextColor(ui.Hex(colorFaint))
		})
		resolutions := [...]int{203, 300, 600}
		selected := max(0, slices.Index(resolutions[:], prefs.DefaultDPI))
		ui.Column(c.Key("picker")).Width(178).Shrink(0).Children(func() {
			segmentedRow(c.Key("segments"), "默认分辨率", []string{"203", "300", "600"}, selected, func(i int) {
				prefs.DefaultDPI = resolutions[i]
				savePreferences()
			})
		})
	})
}

// prefSwitchRow 显示一行带开关的偏好设置
func (e *editor) prefSwitchRow(c *ui.Context, label, hint string, value *bool) {
	ui.Row(c).Height(settingsRowHeight).Padding(0, 16).Gap(12).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(ui.Hex(colorHairline)).Children(func() {
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(ui.Hex(colorBody))
			ui.Text(c, hint).FontSize(10.5).SingleLine().TextColor(ui.Hex(colorFaint))
		})
		before := *value
		ui.Switch(c.Key("switch"), value).Label(label + "开关")
		if before != *value {
			savePreferences()
		}
	})
}

// printSettingsPage 显示当前标签的打印参数与预览入口
func (e *editor) printSettingsPage(c *ui.Context) {
	l := e.label
	ui.Column(c).Gap(0).Children(func() {
		ui.Text(c.Key("title"), "打印").FontSize(20).FontWeight(650).TextColor(ui.Hex(colorHeading))

		ui.Row(c.Key("summary")).Margin(18, 0, 0, 0).Gap(10).Children(func() {
			printSettingCard(c.Key("paper"), iconPaper, "纸张", fmt.Sprintf("%g × %g mm", l.Width, l.Height))
			printSettingCard(c.Key("dpi"), iconPrint, "分辨率", fmt.Sprintf("%d DPI", l.DPI))
			printSettingCard(c.Key("items"), iconStack, "元素", fmt.Sprintf("%d 个", len(l.Items)))
		})

		ui.Column(c.Key("resolution")).Margin(22, 0, 0, 0).Gap(6).Children(func() {
			sectionHeading(c, "打印分辨率")
			ui.Column(c.Key("picker")).Width(300).Children(func() {
				resolutions := [...]int{203, 300, 600}
				selected := max(0, slices.Index(resolutions[:], l.DPI))
				segmentedRow(c.Key("segments"), "打印分辨率", []string{"203 DPI", "300 DPI", "600 DPI"}, selected, func(i int) {
					if l.DPI != resolutions[i] {
						l.DPI = resolutions[i]
						e.changed()
					}
				})
			})
		})

		ui.Column(c.Key("tips")).Margin(24, 0, 0, 0).Gap(7).Children(func() {
			sectionHeading(c, "打印前")
			for _, tip := range []string{
				"先在系统里安装标签打印机驱动，并选择与标签匹配的纸型。",
				"首次打印建议先打一张，测量尺寸后再批量打印。",
				"导出 PDF 后可以用系统预览按 100% 试印。",
			} {
				ui.Row(c.Key(tip)).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Box(c.Key("dot")).Size(4, 4).Radius(2).Shrink(0).Margin(6, 0, 0, 0).
						Background(ui.Hex(colorFaint))
					ui.Text(c, tip).FontSize(11.5).Grow(1).TextColor(ui.Hex(colorMuted))
				})
			}
		})

		ui.Row(c.Key("actions")).Margin(24, 0, 0, 0).Gap(8).Children(func() {
			outlineAction(c.Key("preview"), "打开打印预览", iconPrint, len(l.Items) > 0, func() {
				e.settings.open = false
				e.openPrintPreview()
			})
			outlineAction(c.Key("pdf"), "导出 PDF", iconDownload, len(l.Items) > 0, func() {
				e.settings.open = false
				e.perform(e.exportPDF)
			})
		})
	})
}

// printSettingCard 显示打印设置页的一个信息卡片
func printSettingCard(c *ui.Context, icon *ui.SVG, label, value string) {
	ui.Row(c).Grow(1).Height(64).Padding(0, 12).Gap(10).Radius(10).
		BorderWidth(1).BorderColor(ui.Hex(colorDivider)).Background(ui.Hex(colorFieldBg)).
		AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("icon")).Size(30, 30).Radius(7).Shrink(0).BorderWidth(1).
			BorderColor(ui.Hex(colorFieldBorder)).Background(ui.Hex(colorPanel)).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c.Key("glyph"), icon).Size(15, 15).TextColor(ui.Hex(colorMuted))
		})
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Text(c, label).FontSize(10.5).TextColor(ui.Hex(colorMuted))
			ui.Text(c, value).FontSize(12.5).FontWeight(600).SingleLine().TextColor(ui.Hex(colorText))
		})
	})
}

// aboutPage 显示应用版本和运行环境
func (e *editor) aboutPage(c *ui.Context) {
	ui.Column(c).Gap(0).Children(func() {
		ui.Text(c.Key("title"), "关于").FontSize(20).FontWeight(650).TextColor(ui.Hex(colorHeading))

		ui.Row(c.Key("brand")).Margin(20, 0, 0, 0).Gap(14).AlignItems(ui.Center).Children(func() {
			ui.Row(c.Key("mark")).Size(48, 48).Radius(12).Shrink(0).Background(ui.Hex(colorAccent)).
				Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Icon(c.Key("icon"), iconLogo).Size(26, 26).TextColor(ui.Hex(colorPanel))
			})
			ui.Column(c.Key("text")).Gap(3).Children(func() {
				ui.Text(c, "GoLabel").FontSize(18).FontWeight(650).TextColor(ui.Hex(colorHeading))
				ui.Text(c, "标签编辑器 · 0.1.0").FontSize(11.5).TextColor(ui.Hex(colorMuted))
			})
		})

		ui.Column(c.Key("rows")).Margin(24, 0, 0, 0).Radius(12).BorderWidth(1).
			BorderColor(ui.Hex(colorDivider)).Background(ui.Hex("#FBFCFD")).Children(func() {
			aboutRow(c.Key("version"), "版本", "0.1.0")
			aboutRow(c.Key("runtime"), "运行环境", runtime.Version()+" · "+runtime.GOOS+"/"+runtime.GOARCH)
			aboutRow(c.Key("ui"), "界面框架", "MyGo Native UI 0.8.3")
			aboutRow(c.Key("format"), "模板格式", ".golabel（JSON，含图片）")
			aboutRow(c.Key("license"), "许可", "MIT License")
		})

		ui.Text(c, "所有渲染都在本机完成，标签内容和图片不会上传。").
			FontSize(11).Margin(18, 0, 0, 0).TextColor(ui.Hex(colorMuted))
	})
}

// aboutRow 显示关于页的一行信息
func aboutRow(c *ui.Context, label, value string) {
	ui.Row(c).Height(42).Padding(0, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).Shrink(0).Width(84).TextColor(ui.Hex(colorMuted))
		ui.Text(c, value).FontSize(12).SingleLine().Grow(1).TextColor(ui.Hex(colorText))
	})
}
