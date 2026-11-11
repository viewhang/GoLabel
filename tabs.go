package main

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// documentTabs 管理标签页顺序，并只显示当前文档对应的编辑窗口
type documentTabs struct {
	editors []*editor
	active  *editor
}

// add 注册文档的标签页及其打开、状态变更和关闭事件
func (t *documentTabs) add(e *editor) {
	t.editors = append(t.editors, e)
	e.window.OnShow(func() { t.selectTab(e) })
	e.window.OnClose(func(*mygo.CloseEvent) {
		if t.active != e || len(t.editors) < 2 {
			return
		}
		i := slices.Index(t.editors, e)
		next := i + 1
		if next == len(t.editors) {
			next = i - 1
		}
		matchTabWindow(t.editors[next].window, e.window)
	})
	e.window.OnClosed(func() { t.remove(e) })
	e.document.OnStateChange(func(mygo.DocumentState) {
		if t.active != nil {
			t.active.window.Invalidate()
		}
	})
	e.window.Update(func() { t.selectTab(e) })
}

// matchTabWindow 让下一个标签页沿用当前窗口的位置和大小
func matchTabWindow(target, source *mygo.Window) {
	if source.IsMaximized() {
		if !target.IsMaximized() {
			bounds := source.Bounds()
			target.SetPosition(bounds.X, bounds.Y)
			target.Maximize()
		}
		return
	}
	if target.IsMaximized() {
		target.Unmaximize()
	}
	target.SetBounds(source.Bounds())
}

// selectTab 显示选中的文档并隐藏之前的文档窗口
func (t *documentTabs) selectTab(e *editor) {
	if e == t.active || e.window.IsDestroyed() {
		return
	}
	previous := t.active
	t.active = e
	e.revealTab = true
	if previous != nil {
		matchTabWindow(e.window, previous.window)
	}
	if !e.window.IsVisible() {
		e.window.Show()
	}
	if previous != nil {
		previous.window.Hide()
	}
	e.window.Invalidate()
}

// closeTab 通过文档控制器关闭标签页，有未保存修改时先显示该文档
func (t *documentTabs) closeTab(e *editor) {
	if e.documentEdited() {
		t.selectTab(e)
	}
	e.document.Close()
}

// remove 移除已关闭的文档，并在关闭当前页时切换到相邻页
func (t *documentTabs) remove(e *editor) {
	i := slices.Index(t.editors, e)
	t.editors = slices.Delete(t.editors, i, i+1)
	if t.active != e {
		if t.active != nil {
			t.active.window.Invalidate()
		}
		return
	}
	t.active = nil
	if len(t.editors) > 0 {
		next := t.editors[min(i, len(t.editors)-1)]
		next.window.Update(func() {
			if t.active == nil && len(t.editors) > 0 {
				t.selectTab(t.editors[min(i, len(t.editors)-1)])
			}
		})
	}
}

// tabName 返回标签栏使用的名称，已保存文件省略扩展名
func (e *editor) tabName() string {
	name := e.documentName()
	if e.document != nil && e.document.Path() != "" {
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
				AlignItems(ui.Center).Role(ui.RoleTabList).Label(tr("tabs.label")).Children(func() {
				for _, tab := range e.tabs.editors {
					e.documentTab(c.Key(tab), tab)
				}
				tabIconButton(c.Key("new"), tr("tabs.new"), iconPlus, 28, 14, colors.Muted, func() {
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
			e.window.Update(func() { e.tabs.selectTab(tab) })
		}
		if tab.documentEdited() {
			ui.Box(c.Key("edited")).Size(5, 5).Shrink(0).Margin(0, 2).Radius(3).Background(colors.TabDirty)
		}
		tabIconButton(c.Key("close"), trf("tabs.close", name), iconClose, 20, 10, colors.Faint, func() {
			e.window.Update(func() { e.tabs.closeTab(tab) })
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

// tabStatus 显示当前文档的草稿、保存状态或操作反馈
func (e *editor) tabStatus(c *ui.Context) {
	text, ink, dot := tr("header.saved"), colors.Success, colors.Success
	switch {
	case e.status != "":
		text, ink, dot = e.status, colors.Muted, colors.Faint
	case e.documentEdited():
		text, ink, dot = tr("header.unsaved"), colors.Warning, colors.TabDirty
	case e.document == nil || e.document.Path() == "":
		text, ink, dot = tr("tabs.draft"), colors.Muted, colors.Faint
	}
	ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Box(c.Key("dot")).Size(5, 5).Shrink(0).Radius(3).Background(dot)
		ui.Text(c, text).FontSize(10.5).SingleLine().MaxWidth(180).TextColor(ink)
	})
}
