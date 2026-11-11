package editor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"github.com/pengh/GoLabel/internal/printing"
)

const (
	// panelPadding 是两侧面板的左右内边距
	panelPadding = 16
	// sidebarWidth 是左侧元素与图层面板的宽度
	sidebarWidth = 260
	// inspectorWidth 是右侧属性栏的宽度
	inspectorWidth = 320
)

// editor 保存一个标签页的选择、平移、缩放和交互状态
type editor struct {
	document  *documentFile
	tabs      *documentTabs
	graphics  *platform.Graphics
	printer   *printing.System
	window    *mygo.Window
	label     *label
	selected  int
	zoom      float32
	pan       ui.Point
	canvas    ui.Handle
	drag      *gesture
	textEdit  *textEditState
	inline    bool
	cells     *tableSelection
	status    string // 等待显示的操作反馈
	revealTab bool
	form      newLabelForm
	preview   printPreview
	settings  settingsState
	saveForm  templateSaveForm
}

// documentName 返回当前文件名或新建标签的名称
func (e *editor) documentName() string {
	if e.document != nil && e.document.path != "" {
		return filepath.Base(e.document.path)
	}
	if e.label.title != "" {
		return e.label.title
	}
	name := i18n.Tr("header.untitled")
	if e.document != nil && e.document.untitled > 1 {
		name = fmt.Sprintf("%s %d", name, e.document.untitled)
	}
	return name
}

// documentEdited 报告标签是否有未保存的修改
func (e *editor) documentEdited() bool {
	return e.document != nil && e.document.edited
}

// changed 标记当前标签已修改并安排恢复副本保存
func (e *editor) changed() {
	if e.document != nil {
		e.document.edited = true
		e.document.recoveryDue = time.Now().Add(time.Second)
		e.tabs.syncWindow()
	}
}

// loadLabel 替换文档内容并初始化元素绘制资源和编辑选区
func (e *editor) loadLabel(l *label) {
	e.label, e.selected, e.drag = l, -1, nil
	e.textEdit, e.cells, e.inline = nil, nil, false
	e.form.open, e.preview.open, e.settings.open, e.saveForm.open = false, false, false, false
	for i := range l.Items {
		element := &l.Items[i]
		if e.graphics != nil && (element.Kind == kindQR || element.Kind == kindBarcode) {
			element.prepareSymbol(e.graphics)
		}
	}
}

// view 构建顶部工具栏、左侧元素与图层、画布和右侧属性栏
func (e *editor) view(c *ui.Context) {
	// 每帧按偏好校正配色，「跟随系统」可在系统切换外观后自动跟上
	applyTheme()
	c.SetTheme(appTheme())

	c.OnShortcut(ui.Cmd, ui.KeyP, func() { e.openPrintPreview() })
	c.OnShortcut(ui.Cmd, ui.KeyComma, func() { e.openSettings(settingsGeneral) })
	ui.Column(c).Fill().Background(colors.Canvas).Children(func() {
		e.header(c.Key("header"))
		if e.tabs != nil {
			e.tabBar(c.Key("tabs"))
		}
		ui.Column(c.Key("workspace")).Grow(1).MinHeight(0).Children(func() {
			if e.tabs == nil || len(e.tabs.editors) == 0 {
				e.workspace(c.Key(e), false)
			} else {
				for _, tab := range e.tabs.editors {
					tab.workspace(c.Key(tab), tab != e)
				}
			}
		})
	})
	e.newLabelDialog(c)
	e.printPreviewDialog(c)
	e.settingsDialog(c)
	e.saveTemplateDialog(c.Key("save-template-dialog"))
	if e.status != "" {
		c.Toast(e.status)
		e.status = ""
	}
}

// workspace 保持各标签页的控件状态并仅显示活动标签的编辑区域
func (e *editor) workspace(c *ui.Context, hidden bool) {
	ui.Row(c).Grow(1).MinHeight(0).Hidden(hidden).AlignItems(ui.Stretch).Children(func() {
		e.sidebar(c.Key("sidebar"))
		e.canvasView(c.Key("canvas"))
		e.inspector(c.Key("inspector"))
	})
}

// header 显示品牌、文件操作、导出和打印按钮
func (e *editor) header(c *ui.Context) {
	l := e.label
	ui.Row(c).Height(62).Shrink(0).Padding(0, 20).Gap(16).AlignItems(ui.Center).
		Background(colors.Header).BorderWidth(0, 0, 1, 0).BorderColor(colors.Divider).
		Children(func() {
			ui.Row(c.Key("brand")).Gap(10).Shrink(0).AlignItems(ui.Center).Children(func() {
				ui.Row(c.Key("mark")).Size(30, 30).Radius(8).Shrink(0).Background(colors.Primary).
					Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconLogo).Size(17, 17).TextColor(colors.OnAccent)
				})
				ui.Text(c, "GoLabel").FontSize(14.5).FontWeight(650).TextColor(colors.Heading)
			})
			ui.Box(c.Key("rule")).Width(1).Height(20).Shrink(0).Background(colors.Divider)

			ui.Row(c.Key("file")).Gap(1).Shrink(0).Children(func() {
				ghostAction(c.Key("new"), i18n.Tr("header.new"), iconPlus, func() { e.form.openNewLabel() })
				ghostAction(c.Key("open"), i18n.Tr("header.open"), iconFolder, func() {
					e.perform(e.tabs.openDialog)
				})
				fileMenuButton(c.Key("recent"), i18n.Tr("header.recent"), e.recentFilesMenu)
				ghostAction(c.Key("save"), i18n.Tr("header.save"), iconSave, func() {
					e.perform(e.save)
				})
				fileMenuButton(c.Key("save-options"), i18n.Tr("header.saveOptions"), func(m *ui.Menu) {
					if m.Item(i18n.Tr("template.saveAs")).Chosen() {
						e.openSaveTemplate()
					}
				})
			})

			ui.Box(c.Key("spacer")).Grow(1)
			empty := len(l.Items) == 0
			ui.Row(c.Key("actions")).Gap(8).Shrink(0).AlignItems(ui.Center).Children(func() {
				ghostAction(c.Key("templates"), i18n.Tr("header.templates"), iconGrid, func() { e.openSettings(settingsTemplates) })
				ghostAction(c.Key("settings"), i18n.Tr("header.settings"), iconSliders, func() { e.openSettings(settingsGeneral) })
				outlineAction(c.Key("pdf"), i18n.Tr("header.export"), iconDownload, !empty, func() {
					e.perform(e.exportPDF)
				})
				accentAction(c.Key("print"), i18n.Tr("header.print"), iconPrint, !empty, func() {
					e.openPrintPreview()
				})
			})
		})
}

// fileMenuButton 显示文件操作旁的下拉菜单按钮
func fileMenuButton(c *ui.Context, label string, build func(*ui.Menu)) {
	button := ui.ButtonBase(c).Size(22, 32).Shrink(0).Radius(6).
		AlignItems(ui.Center).Justify(ui.Center).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.Background(colors.Hover)
	}
	button.Children(func() {
		ui.Icon(c, iconChevron).Size(13, 13).Rotate(90).TextColor(colors.Body)
	})
	button.PopoverMenuButton(build)
}

// recentFilesMenu 列出最近打开或保存的标签并按选中路径打开
func (e *editor) recentFilesMenu(m *ui.Menu) {
	paths := mygo.App.RecentDocuments()
	if len(paths) == 0 {
		m.Item(i18n.Tr("header.noRecent")).Disabled(true)
		return
	}
	m.MaxWidth(420)
	for _, path := range paths {
		if m.Item(filepath.Base(path)).ID(path).Subtitle(filepath.Dir(path)).Tooltip(path).Chosen() {
			e.perform(func() error {
				return e.tabs.open(path)
			})
		}
	}
}

// ghostAction 显示顶栏中无边框的图标按钮
func ghostAction(c *ui.Context, label string, icon *ui.SVG, action func()) {
	button := ui.ButtonBase(c).Height(32).Padding(0, 9).Radius(7).Gap(6).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.Background(colors.Hover)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(colors.Body)
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(colors.Body)
	})
	if button.Clicked() {
		action()
	}
}

// outlineStyle 描述描边按钮使用的一套颜色
type outlineStyle struct {
	ink, border, fill, hover, hoverBorder ui.Color
}

// outlineAction 显示顶栏中描边的次要按钮
func outlineAction(c *ui.Context, label string, icon *ui.SVG, enabled bool, action func()) {
	style := outlineStyle{
		ink: colors.Body, border: colors.FieldBorder, fill: colors.Panel,
		hover: colors.FieldBg, hoverBorder: colors.BorderHover,
	}
	if !enabled {
		style.ink, style.border, style.fill = colors.Disabled, colors.Divider, colors.FieldBg
	}
	button := ui.ButtonBase(c).Height(34).Padding(0, 12).Radius(8).Gap(6).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Disabled(!enabled).
		BorderWidth(1).BorderColor(style.border).Background(style.fill)
	if button.Hovered() && enabled {
		button.Background(style.hover).BorderColor(style.hoverBorder)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(style.ink)
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(style.ink)
	})
	if button.Clicked() && enabled {
		action()
	}
}

// accentAction 显示顶栏中的主色按钮
func accentAction(c *ui.Context, label string, icon *ui.SVG, enabled bool, action func()) {
	fill, ink := colors.Primary, colors.OnAccent
	if !enabled {
		fill, ink = colors.Badge, colors.Disabled
	}
	button := ui.ButtonBase(c).Height(34).Padding(0, 14).Radius(8).Gap(7).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Disabled(!enabled).
		Background(fill).Shadow(0, 1, 3, 0, colors.ShadowMid)
	if button.Hovered() && enabled {
		button.Background(colors.PrimaryHover)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(ink)
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(ink)
	})
	if button.Clicked() && enabled {
		action()
	}
}

// sidebar 显示添加元素、图层列表和纸张设置
func (e *editor) sidebar(c *ui.Context) {
	l := e.label
	ui.Column(c).Width(sidebarWidth).Shrink(0).FillHeight().Background(colors.Panel).
		BorderWidth(0, 1, 0, 0).BorderColor(colors.Divider).Children(func() {
		ui.Row(c.Key("add-head")).Padding(18, panelPadding, 10, panelPadding).Shrink(0).Children(func() {
			sectionHeading(c, i18n.Tr("sidebar.add"))
		})
		ui.Scroll(c.Key("tools")).MaxHeightPercent(55).MinHeight(0).Shrink(0).Children(func() {
			ui.Column(c.Key("cards")).Padding(0, panelPadding, 16, panelPadding).Gap(8).Children(func() {
				tools := [...]struct {
					kind itemKind
					name string
					icon *ui.SVG
				}{
					{kindText, i18n.Tr("kind.text"), iconText}, {kindImage, i18n.Tr("kind.image"), iconPicture},
					{kindBarcode, i18n.Tr("kind.barcode"), iconBarcode}, {kindQR, i18n.Tr("kind.qr"), iconQR},
					{kindLine, i18n.Tr("sidebar.line"), iconLine}, {kindTable, i18n.Tr("kind.table"), iconTable},
					{kindRectangle, i18n.Tr("kind.rectangle"), iconRectangle}, {kindEllipse, i18n.Tr("kind.ellipse"), iconEllipse},
					{kindDiamond, i18n.Tr("kind.diamond"), iconDiamond}, {kindTriangle, i18n.Tr("kind.triangle"), iconTriangle},
					{kindStar, i18n.Tr("kind.star"), iconStar}, {kindHexagon, i18n.Tr("kind.hexagon"), iconHexagon},
				}
				for i := 0; i < len(tools); i += 2 {
					ui.Row(c.Key(i)).Gap(8).Children(func() {
						for _, tool := range tools[i:min(i+2, len(tools))] {
							e.toolCard(c.Key(tool.kind), tool.kind, tool.name, tool.icon)
						}
					})
				}
			})
		})

		hairline(c.Key("divider"))

		ui.Row(c.Key("layer-head")).Padding(14, panelPadding, 8, panelPadding).Gap(7).Shrink(0).AlignItems(ui.Center).Children(func() {
			sectionHeading(c, i18n.Tr("sidebar.layers"))
			ui.Row(c.Key("count")).Height(17).MinWidth(17).Padding(0, 5).Radius(9).Shrink(0).
				Background(colors.Hover).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d", len(l.Items)).FontSize(10).FontWeight(600).TextColor(colors.Muted)
			})
		})

		ui.Scroll(c.Key("layers")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key("rows")).Padding(0, 8, 8, 8).Gap(2).Children(func() {
				if len(l.Items) == 0 {
					emptyLayers(c.Key("empty"))
					return
				}
				for i := len(l.Items) - 1; i >= 0; i-- {
					e.layerRow(c.Key(i), i)
				}
			})
		})

		ui.Column(c.Key("paper-settings")).Padding(8, 14, 14, 14).Shrink(0).Children(func() {
			e.paperSettings(c.Key("paper"))
		})
	})
}

// toolCard 显示可点击或拖入画布的元素工具卡片
func (e *editor) toolCard(c *ui.Context, kind itemKind, name string, icon *ui.SVG) {
	// 图标比文字淡一档，悬停时点亮成主色
	ink := colors.Muted
	card := ui.ButtonBase(c).Grow(1).Height(60).Radius(10).Column().Gap(7).
		AlignItems(ui.Center).Justify(ui.Center).Role(ui.RoleButton).Label(name).
		Cursor(ui.CursorPointer).Drag(kind).
		BorderWidth(1).BorderColor(colors.CardBorder).Background(colors.Tile)
	if card.Hovered() {
		card.BorderColor(colors.BorderHover).Background(colors.TileHover)
		ink = colors.Accent
	}
	card.Children(func() {
		ui.Icon(c, icon).Size(19, 19).TextColor(ink)
		ui.Text(c, name).FontSize(12).FontWeight(500).TextColor(colors.Body)
	})
	if x, y, ok := e.toolPosition(c, card); ok {
		if kind == kindImage {
			e.perform(func() error { return e.importImage(x, y) })
		} else {
			e.add(item{Kind: kind, X: x, Y: y})
		}
	}
}

// toolPosition 将添加工具的点击或拖放落点换算为标签毫米坐标
func (e *editor) toolPosition(c *ui.Context, button ui.Element) (x, y float64, ok bool) {
	if button.Clicked() {
		return 2, 2, true
	}
	point, dropped := button.DragMissed()
	if !dropped {
		return 0, 0, false
	}
	bounds := e.canvas.Bounds(c)
	paper := e.paperRect(bounds)
	if !bounds.Contains(point.X, point.Y) || !paper.Contains(point.X, point.Y) {
		return 0, 0, false
	}
	x = roundMM(float64(point.X-paper.X) / float64(paper.W) * e.label.Width)
	y = roundMM(float64(point.Y-paper.Y) / float64(paper.H) * e.label.Height)
	return x, y, true
}

// layerRow 显示一个图层的图标、名称和毫米尺寸
func (e *editor) layerRow(c *ui.Context, index int) {
	a := &e.label.Items[index]
	selected := index == e.selected
	ink, name, size := colors.Faint, colors.Body, colors.Faint
	chip, chipBorder := colors.LayerChip, colors.Hairline
	if selected {
		ink, name, size = colors.Accent, colors.Text, colors.Muted
		chip, chipBorder = colors.Panel, colors.FieldBorder
	}
	weight := 500
	if selected {
		weight = 600
	}
	row := ui.ButtonBase(c).Height(38).FillWidth().Padding(0, 8).Radius(7).Gap(9).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(layerName(a)).AccessSelected(selected)
	switch {
	case selected:
		row.Background(colors.Selected)
	case row.Hovered():
		row.Background(colors.LayerHover)
	}
	row.Children(func() {
		ui.Row(c.Key("icon")).Size(22, 22).Radius(5).Shrink(0).BorderWidth(1).
			BorderColor(chipBorder).Background(chip).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, layerIcon(a.Kind)).Size(13, 13).TextColor(ink)
		})
		ui.Text(c, layerName(a)).FontSize(12.5).FontWeight(weight).SingleLine().
			TextColor(name).Grow(1)
		ui.Text(c, fmt.Sprintf("%g × %g", a.W, a.H)).FontSize(10).Shrink(0).TextColor(size)
	})
	if row.Clicked() {
		e.selectItem(index)
	}
}

// layerName 返回图层列表中显示的元素名称
func layerName(a *item) string {
	if a.Kind == kindText {
		if name := []rune(a.textContent().PlainText()); len(name) > 0 {
			return string(name[:min(12, len(name))])
		}
		return i18n.Tr("kind.text")
	}
	return kindTitle(a.Kind)
}

// emptyLayers 在没有元素时提示从左侧添加
func emptyLayers(c *ui.Context) {
	ui.Column(c).Margin(0, 6, 0, 0).Padding(26, 14).Radius(10).Gap(10).
		BorderWidth(1).BorderStyle(ui.BorderDashed).BorderColor(colors.DashBorder).
		Background(colors.FieldBg).AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("badge")).Size(34, 34).Radius(8).Background(colors.ChipAlt).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconStack).Size(17, 17).TextColor(colors.RulerText)
		})
		ui.Text(c, i18n.Tr("sidebar.empty")).FontSize(11.5).FontWeight(600).TextColor(colors.Body2)
	})
}

// paperSettings 以规格卡片打开纸张尺寸和 DPI 设置
func (e *editor) paperSettings(c *ui.Context) {
	l := e.label
	button := ui.ButtonBase(c).Height(52).FillWidth().Padding(0, 11).Gap(10).Radius(10).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(i18n.Tr("sidebar.paper")).Cursor(ui.CursorPointer).
		BorderWidth(1).BorderColor(colors.CardBorder).Background(colors.Tile)
	if button.Hovered() {
		button.Background(colors.TileHover).BorderColor(colors.BorderHover)
	}
	button.Children(func() {
		ui.Row(c.Key("icon")).Size(30, 30).Radius(7).Shrink(0).BorderWidth(1).
			BorderColor(colors.FieldBorder).Background(colors.Panel).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPaper).Size(16, 16).TextColor(colors.Body2)
		})
		ui.Column(c.Key("text")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Start).Children(func() {
			ui.Textf(c, "%g × %g mm", l.Width, l.Height).FontSize(12).FontWeight(600).TextColor(colors.Text)
			ui.Textf(c, "%d DPI", l.DPI).FontSize(10).TextColor(colors.Muted)
		})
		ui.Icon(c, iconChevron).Size(14, 14).Shrink(0).TextColor(colors.Faint)
	})

	open := ui.Local(button, "open", func() bool { return false })
	if button.Clicked() {
		*open = !*open
	}
	ui.Popover(c.Key("popover"), button, open, func() {
		ui.Column(c.Key("fields")).Width(228).Padding(12).Gap(10).Children(func() {
			w, h, dpi := l.Width, l.Height, l.DPI
			ui.Row(c.Key("paper-size")).Gap(8).Children(func() {
				inspectorDimension(c, i18n.Tr("sidebar.paperWidth"), &l.Width, minimumSide, maximumSide)
				inspectorDimension(c, i18n.Tr("sidebar.paperHeight"), &l.Height, minimumSide, maximumSide)
			})
			if w != l.Width || h != l.Height {
				for i := range l.Items {
					l.Items[i].constrain(l)
				}
			}

			resolutions := [...]int{203, 300, 600}
			selectedDPI := slices.Index(resolutions[:], l.DPI)
			ui.Column(c.Key("dpi")).Gap(5).Children(func() {
				ui.Text(c, i18n.Tr("sidebar.dpi")).FontSize(10).TextColor(colors.Muted)
				optionRow(c.Key("segments"), []string{"203", "300", "600"}, selectedDPI, func(i int) {
					l.DPI = resolutions[i]
				})
			})
			if w != l.Width || h != l.Height || dpi != l.DPI {
				e.changed()
			}
		})
	})
}

// add 添加带默认尺寸的元素并选中它
func (e *editor) add(a item) {
	switch a.Kind {
	case kindText:
		a.W, a.H, a.FontSize, a.Value = 24, 6, 4, i18n.Tr("default.text")
	case kindQR:
		a.W, a.H, a.Value = 18, 18, "GoLabel"
	case kindBarcode:
		a.W, a.H, a.Value = 32, 9, "12345678"
	case kindImage:
		a.W, a.H = 18, 18
	case kindLine:
		a.W, a.H, a.Stroke = 28, 3, 0.3
	case kindRectangle:
		a.W, a.H, a.Stroke = 24, 14, 0.3
	case kindEllipse, kindDiamond, kindTriangle, kindStar, kindHexagon:
		a.W, a.H, a.Stroke = 16, 16, 0.3
	case kindTable:
		a.W, a.H, a.Rows, a.Columns, a.Stroke = 30, 16, 3, 3, 0.3
	}
	if a.Kind == kindQR || a.Kind == kindBarcode {
		a.prepareSymbol(e.graphics)
	}
	a.constrain(e.label)
	e.label.Items = append(e.label.Items, a)
	e.selectItem(len(e.label.Items) - 1)
	e.changed()
}

// remove 删除当前选中的标签元素
func (e *editor) remove() {
	e.label.Items = slices.Delete(e.label.Items, e.selected, e.selected+1)
	e.selected, e.drag = -1, nil
	e.textEdit, e.cells, e.inline = nil, nil, false
	e.changed()
}

// perform 将文件操作和系统对话框排入窗口事件队列
func (e *editor) perform(action func() error) {
	e.window.Update(func() {
		if err := action(); err != nil && !errors.Is(err, mygo.ErrCanceled) {
			target := e
			if e.tabs != nil {
				target = e.tabs.active
			}
			target.status = err.Error()
		}
	})
}

// importImage 打开图片并将原图内嵌到标签的指定位置
func (e *editor) importImage(x, y float64) error {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: e.window, Title: i18n.Tr("dialog.importTitle"),
		Filters: []mygo.FileFilter{{Name: i18n.Tr("dialog.importFilter"), Extensions: []string{"png", "jpg", "jpeg"}}},
	})
	if err != nil || len(paths) == 0 {
		return err
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return err
	}
	bitmap, err := decodeImage(data)
	if err != nil {
		return err
	}
	e.add(item{Kind: kindImage, X: x, Y: y, Data: data, bitmap: bitmap})
	return nil
}

// exportPDF 将当前标签保存为实际尺寸的 PDF
func (e *editor) exportPDF() error {
	page, err := e.label.printPage()
	if err != nil {
		return err
	}
	data, err := e.graphics.PDF(page)
	if err != nil {
		return err
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: e.window, Title: i18n.Tr("dialog.exportTitle"), DefaultPath: i18n.Tr("dialog.exportName"),
		Filters: []mygo.FileFilter{{Name: "PDF", Extensions: []string{"pdf"}}},
	})
	if err != nil || path == "" {
		return err
	}
	if err := platform.WritePDFFile(path, data); err != nil {
		return err
	}
	e.status = i18n.Tr("status.exported")
	return nil
}

// print 打开系统打印面板并显示提交结果
func (e *editor) print() error {
	page, err := e.label.printPage()
	if err != nil {
		return err
	}
	sent, err := e.printer.Submit(printing.Job{Page: page, Title: e.documentName()}, e.window.NativeHandle())
	if sent {
		e.status = i18n.Tr("status.printed")
	}
	return err
}
