package main

import (
	"log"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
)

// main 注册标签模板和 MyGo 原生编辑窗口
func main() {
	graphics, err := openGraphics()
	if err != nil {
		log.Fatal(err)
	}
	mygo.App.SetName("GoLabel")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// 应用自己的语言排在前面，系统语言都不匹配时用它
	mygo.App.SetLanguages(string(languageZhHans), string(languageEnglish))
	// 原生菜单和系统对话框跟随界面语言：选定语言时告诉框架，跟随系统时交回给系统
	appLocale = func(name string) {
		if language(name) == languageSystem {
			mygo.App.SetLocale("")
			return
		}
		mygo.App.SetLocale(name)
	}
	loadPreferences()
	applyLanguage()
	applyTheme()
	loadUserTemplates()
	// 原生菜单要等应用启动后再切换语言，那时系统接口才可用
	mygo.App.WhenReady(func() { appLocale(prefs.Language) })
	tabs := &documentTabs{}
	var documents *mygo.Documents[*label]
	documents = mygo.NewDocuments(mygo.DocumentsOptions[*label]{
		Types: []mygo.DocumentType{{Name: tr("app.templateType"), Extensions: []string{"golabel"}}},
		New: func() *label {
			// 新建对话框选好规格后先暂存，文档控制器创建时取用
			if l := nextLabel.Swap(nil); l != nil {
				return l
			}
			return newLabel()
		},
		Read:      readLabel,
		Write:     writeLabel,
		Snapshot:  (*label).snapshot,
		Autosave:  mygo.AutosaveOff,
		AskOnQuit: true,
		Window: func(document *mygo.Document[*label]) *mygo.Window {
			e := &editor{document: document, documents: documents, graphics: graphics, tabs: tabs, selected: -1, zoom: 1}
			e.window = mygo.NewWindow(mygo.WindowOptions{
				Title: "GoLabel", Width: 1280, Height: 800, MinWidth: 980, MinHeight: 660,
				Content: ui.View(e.view), Hidden: true,
			})
			tabs.add(e)
			return e.window
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
