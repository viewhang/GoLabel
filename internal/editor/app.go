package editor

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/platform"
	"github.com/pengh/GoLabel/internal/printing"
)

// Run 创建单个原生主窗口并运行多标签文档编辑器
func Run() error {
	graphics, err := platform.OpenGraphics()
	if err != nil {
		return err
	}
	printer, err := printing.OpenSystem(graphics.PDF)
	if err != nil {
		return err
	}
	mygo.App.SetName("GoLabel")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	tabs := &documentTabs{graphics: graphics, printer: printer}
	mygo.App.SetLanguages(string(languageZhHans), string(languageEnglish))
	mygo.App.OnLocaleChanged(func(string) {
		resetFontMenu()
		if tabs.window != nil {
			tabs.installMenu()
			tabs.syncWindow()
		}
	})
	loadPreferences()
	applyLanguage()
	setNativeTheme = mygo.Theme.SetSource
	applyThemePreference()
	loadUserTemplates()
	mygo.App.WhenReady(tabs.openWindow)
	mygo.App.OnOpenFile(func(path string) {
		if !strings.EqualFold(filepath.Ext(path), ".golabel") {
			return
		}
		mygo.App.WhenReady(func() {
			tabs.active.perform(func() error {
				if err := tabs.open(path); err != nil {
					return err
				}
				tabs.window.Show()
				tabs.window.Restore()
				tabs.window.Focus()
				return nil
			})
		})
	})
	mygo.App.OnSecondInstance(func([]string, string) {
		mygo.App.WhenReady(func() {
			tabs.window.Restore()
			tabs.window.Focus()
		})
	})
	return mygo.App.Run()
}

// openWindow 在应用就绪后创建唯一主窗口并恢复未保存标签
func (tabs *documentTabs) openWindow() {
	tabs.initial = tabs.newTab(newLabel())
	tabs.window = mygo.NewWindow(mygo.WindowOptions{
		Title: "GoLabel", Width: 1280, Height: 800, MinWidth: 980, MinHeight: 660,
		Content: ui.View(tabs.view), Hidden: true,
	})
	tabs.active.window = tabs.window
	tabs.window.OnClose(tabs.closeWindow)
	tabs.window.OnFocus(func() {
		for _, e := range tabs.editors {
			e.flushTextEdit()
			if e.document.path != "" {
				e.checkFile()
			}
		}
	})
	tabs.restoreRecovery()
	tabs.installMenu()
	tabs.syncWindow()
	tabs.window.Show()
}
