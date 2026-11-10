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
	mygo.App.SetLanguages("zh-Hans")
	var documents *mygo.Documents[*label]
	documents = mygo.NewDocuments(mygo.DocumentsOptions[*label]{
		Types:     []mygo.DocumentType{{Name: "GoLabel 标签模板", Extensions: []string{"golabel"}}},
		New:       newLabel,
		Read:      readLabel,
		Write:     writeLabel,
		Snapshot:  (*label).snapshot,
		Autosave:  mygo.AutosaveOff,
		AskOnQuit: true,
		Window: func(document *mygo.Document[*label]) *mygo.Window {
			e := &editor{document: document, documents: documents, graphics: graphics, selected: -1, zoom: 1}
			e.window = mygo.NewWindow(mygo.WindowOptions{
				Title: "GoLabel", Width: 1100, Height: 740, MinWidth: 800, MinHeight: 630,
				Content: ui.View(e.view),
			})
			return e.window
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
