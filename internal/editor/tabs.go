package editor

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"github.com/pengh/GoLabel/internal/printing"
)

// documentTabs 在一个主窗口中管理文档及各标签页的编辑状态
type documentTabs struct {
	editors      []*editor
	active       *editor
	initial      *editor
	window       *mygo.Window
	graphics     *platform.Graphics
	printer      *printing.System
	nextUntitled int
	closing      bool
}

// view 在固定主窗口中构建当前标签的公共界面
func (t *documentTabs) view(c *ui.Context) {
	if t.active != nil {
		t.active.view(c)
		t.checkpoint(c)
	}
}

// newTab 创建独立文档状态并在当前主窗口中选中标签
func (t *documentTabs) newTab(l *label) *editor {
	t.initial = nil
	t.nextUntitled++
	e := &editor{tabs: t, window: t.window, graphics: t.graphics, printer: t.printer,
		document: &documentFile{untitled: t.nextUntitled}, selected: -1, zoom: 1}
	e.loadLabel(l)
	t.editors = append(t.editors, e)
	t.selectTab(e)
	return e
}

// removeInitial 移除打开文件前尚未使用的启动空白标签
func (t *documentTabs) removeInitial(e *editor) {
	if !e.documentEdited() && len(e.label.Items) == 0 {
		t.editors = slices.DeleteFunc(t.editors, func(tab *editor) bool { return tab == e })
	}
	t.initial = nil
}

// selectTab 写回离开标签的文字并切换活动文档
func (t *documentTabs) selectTab(e *editor) {
	if e == t.active {
		return
	}
	if t.active != nil {
		t.active.flushTextEdit()
	}
	t.active = e
	e.revealTab = true
	t.syncWindow()
}

// syncWindow 将当前标签的文件名和保存状态同步到主窗口
func (t *documentTabs) syncWindow() {
	if t.window == nil || t.active == nil {
		return
	}
	e := t.active
	t.window.SetTitle(e.documentName())
	t.window.SetRepresentedFile(e.document.path)
	t.window.SetDocumentEdited(e.document.edited)
	t.window.Invalidate()
}

// closeTab 确认未保存修改后移除标签并选中相邻标签
func (t *documentTabs) closeTab(e *editor) {
	if t.closing {
		return
	}
	t.closing = true
	defer func() { t.closing = false }()
	if !t.confirmClose(e) {
		return
	}
	i := slices.Index(t.editors, e)
	e.stopWatching()
	e.removeRecovery()
	t.editors = slices.Delete(t.editors, i, i+1)
	if e == t.active {
		t.active = nil
		if len(t.editors) == 0 {
			t.newTab(newLabel())
		} else {
			t.selectTab(t.editors[min(i, len(t.editors)-1)])
		}
	}
	t.syncWindow()
}

// tabName 返回标签栏使用的名称，已保存文件省略扩展名
func (e *editor) tabName() string {
	name := e.documentName()
	if e.document != nil && e.document.path != "" {
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
	return name
}

// tabBar 显示文档标签、新建入口和当前文档的保存状态
func (e *editor) tabBar(c *ui.Context) {
	ui.Row(c).Height(42).Shrink(0).Padding(0, 14).Gap(8).AlignItems(ui.Center).
		Background(colors.TabBar).BorderWidth(0, 0, 1, 0).BorderColor(colors.Divider).
		Children(func() {
			ui.ScrollHorizontal(c.Key("strip")).Grow(1).MinWidth(0).Height(42).Gap(4).
				AlignItems(ui.Center).Role(ui.RoleTabList).Label(i18n.Tr("tabs.label")).Children(func() {
				for _, tab := range e.tabs.editors {
					e.documentTab(c.Key(tab), tab)
				}
				tabIconButton(c.Key("new"), i18n.Tr("tabs.new"), iconPlus, 28, 14, colors.Muted, func() {
					e.form.openNewLabel()
				})
			})
			e.tabStatus(c.Key("status"))
		})
}

// documentTab 显示单个文档的名称、未保存标记和关闭按钮
func (e *editor) documentTab(c *ui.Context, tab *editor) {
	selected := tab == e
	name := tab.tabName()
	ink, iconInk, weight := colors.Muted, colors.Faint, 500
	row := ui.Row(c).Height(30).Shrink(0).Padding(0, 4, 0, 8).Gap(2).Radius(7).
		AlignItems(ui.Center).BorderWidth(1).BorderColor(ui.Color{})
	if selected {
		ink, iconInk, weight = colors.Text, colors.Accent, 600
		row.Background(colors.Floating).BorderColor(colors.FieldBorder).Shadow(0, 1, 2, 0, colors.ShadowSoft)
		if e.revealTab {
			row.ScrollIntoView()
			e.revealTab = false
		}
	} else if row.Hovered() {
		row.Background(colors.Hover)
	}
	row.Children(func() {
		button := ui.ButtonBase(c.Key("select")).Height(28).Gap(6).AlignItems(ui.Center).
			Role(ui.RoleTab).Label(name).Tooltip(name).AccessSelected(selected).Cursor(ui.CursorPointer)
		button.Children(func() {
			ui.Icon(c, iconLogo).Size(13, 13).Shrink(0).TextColor(iconInk)
			ui.Text(c, name).FontSize(12).FontWeight(weight).SingleLine().MaxWidth(160).TextColor(ink)
		})
		if button.Clicked() && !selected {
			e.tabs.selectTab(tab)
		}
		if tab.documentEdited() {
			ui.Box(c.Key("edited")).Size(5, 5).Shrink(0).Margin(0, 2).Radius(3).Background(colors.TabDirty)
		}
		tabIconButton(c.Key("close"), i18n.Trf("tabs.close", name), iconClose, 20, 10, colors.Faint, func() {
			e.perform(func() error { e.tabs.closeTab(tab); return nil })
		})
	})
}

// tabIconButton 绘制标签栏中的图标按钮
func tabIconButton(c *ui.Context, label string, icon *ui.SVG, size, iconSize float32, ink ui.Color, action func()) {
	button := ui.ButtonBase(c).Size(size, size).Shrink(0).Radius(5).
		AlignItems(ui.Center).Justify(ui.Center).Role(ui.RoleButton).Label(label).Tooltip(label).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.Background(colors.Hover)
		ink = colors.Body
	}
	button.Children(func() { ui.Icon(c, icon).Size(iconSize, iconSize).TextColor(ink) })
	if button.Clicked() {
		action()
	}
}

// tabStatus 显示当前文档的保存状态
func (e *editor) tabStatus(c *ui.Context) {
	var state documentFile
	if e.document != nil {
		state = *e.document
	}
	text, ink, dot := i18n.Tr("header.saved"), colors.Success, colors.Success
	switch {
	case state.deleted:
		text, ink, dot = i18n.Tr("tabs.deleted"), colors.Warning, colors.TabDirty
	case state.edited:
		text, ink, dot = i18n.Tr("header.unsaved"), colors.Warning, colors.TabDirty
	case state.path == "":
		text, ink, dot = i18n.Tr("tabs.draft"), colors.Muted, colors.Faint
	}
	ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Box(c.Key("dot")).Size(5, 5).Shrink(0).Radius(3).Background(dot)
		ui.Text(c, text).FontSize(10.5).SingleLine().MaxWidth(180).TextColor(ink)
	})
}
