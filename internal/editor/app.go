package editor

import (
	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
	"github.com/pengh/GoLabel/internal/printing"
)

// Run 注册标签文档和 MyGo 原生编辑窗口，再运行应用
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
	// 应用自己的语言排在前面，系统语言都不匹配时用它
	mygo.App.SetLanguages(string(languageZhHans), string(languageEnglish))
	mygo.App.OnLocaleChanged(func(string) { resetFontMenu() })
	loadPreferences()
	applyLanguage()
	setNativeTheme = mygo.Theme.SetSource
	applyThemePreference()
	loadUserTemplates()
	tabs := &documentTabs{}
	mygo.App.OnSecondInstance(func(args []string, _ string) {
		// 文件参数由 MyGo 文档控制器打开，普通启动恢复当前标签窗口
		if len(args) == 0 && tabs.active != nil {
			window := tabs.active.window
			window.Restore()
			window.Focus()
		}
	})
	var documents *mygo.Documents[*label]
	documents = mygo.NewDocuments(mygo.DocumentsOptions[*label]{
		Types: []mygo.DocumentType{{Name: i18n.Tr("app.templateType"), Extensions: []string{"golabel"}}},
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
			e := &editor{document: document, documents: documents, graphics: graphics, printer: printer, tabs: tabs, selected: -1, zoom: 1}
			e.window = mygo.NewWindow(mygo.WindowOptions{
				Title: "GoLabel", Width: 1280, Height: 800, MinWidth: 980, MinHeight: 660,
				Content: ui.View(e.view), Hidden: true,
			})
			tabs.add(e)
			return e.window
		},
	})
	return mygo.App.Run()
}
