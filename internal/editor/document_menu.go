package editor

import (
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/pengh/GoLabel/internal/i18n"
)

// installMenu 将文件菜单和快捷键绑定到主窗口的活动标签
func (t *documentTabs) installMenu() {
	var recent []*mygo.MenuItem
	for _, path := range mygo.App.RecentDocuments() {
		recent = append(recent, &mygo.MenuItem{
			Label: filepath.Base(path), ToolTip: path,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.perform(func() error { return t.open(path) })
			},
		})
	}
	if len(recent) > 0 {
		recent = append(recent, mygo.Separator(), &mygo.MenuItem{
			Label: i18n.Tr("document.clearRecent"), Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.perform(func() error {
					if err := mygo.App.ClearRecentDocuments(); err != nil {
						return err
					}
					t.installMenu()
					return nil
				})
			},
		})
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: i18n.Tr("document.fileMenu"), Submenu: []*mygo.MenuItem{
			{Label: i18n.Tr("header.new"), Accelerator: "CmdOrCtrl+N", Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.form.openNewLabel()
				t.window.Invalidate()
			}},
			{Label: i18n.Tr("header.open"), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.perform(t.openDialog)
			}},
			{Label: i18n.Tr("header.recent"), Submenu: recent, Disabled: len(recent) == 0},
			mygo.Separator(),
			{Label: i18n.Tr("header.save"), Accelerator: "CmdOrCtrl+S", Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.perform(t.active.save)
			}},
			{Label: i18n.Tr("document.saveAs"), Accelerator: "CmdOrCtrl+Shift+S", Click: func(*mygo.MenuItem, *mygo.Window) {
				t.active.perform(t.active.saveAs)
			}},
			{Label: i18n.Tr("document.closeTab"), Accelerator: "CmdOrCtrl+W", Click: func(*mygo.MenuItem, *mygo.Window) {
				target := t.active
				target.perform(func() error { t.closeTab(target); return nil })
			}},
			{Role: mygo.RoleQuit, Hidden: runtime.GOOS == "darwin"},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}))
}
