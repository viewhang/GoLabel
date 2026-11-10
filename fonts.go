package main

import (
	"cmp"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/ui"
)

const (
	// defaultFontName 是字体选择器中系统默认字体的名称
	defaultFontName = "系统默认"
	// sourceHanFamily 是思源黑体中国大陆版的字体族名
	sourceHanFamily = "Source Han Sans CN"
)

// fontMenu 保存字体菜单显示名称与模板字体族名的对应关系
type fontMenu struct {
	options      []string
	nameByFamily map[string]string
	familyByName map[string]string
}

// labelFonts 返回带本地化名称的可用字体菜单
var labelFonts = sync.OnceValue(func() fontMenu {
	defer macPool()()
	manager := macSend[objc.ID](macClass("NSFontManager"), "sharedFontManager")
	menu := fontMenu{
		options:      []string{defaultFontName},
		nameByFamily: map[string]string{"": defaultFontName},
		familyByName: map[string]string{defaultFontName: ""},
	}
	for _, font := range ui.FontFamilies() {
		name := font.Name
		if font.Name == sourceHanFamily {
			name = "思源黑体"
		} else {
			local := macSend[objc.ID](manager, "localizedNameForFamily:face:", macString(font.Name), objc.ID(0))
			name = cmp.Or(macSend[string](local, "UTF8String"), name)
		}
		if _, exists := menu.familyByName[name]; exists {
			name += "（" + font.Name + "）"
		}
		menu.options = append(menu.options, name)
		menu.nameByFamily[font.Name], menu.familyByName[name] = name, font.Name
	}
	return menu
})
