package main

import (
	"sync"

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

// labelFonts 缓存系统字体菜单
var labelFonts = sync.OnceValue(systemFontMenu)

// makeFontMenu 枚举系统字体并建立显示名和字体族名的对应关系
func makeFontMenu(displayName func(string) string) fontMenu {
	menu := fontMenu{
		options:      []string{defaultFontName},
		nameByFamily: map[string]string{"": defaultFontName},
		familyByName: map[string]string{defaultFontName: ""},
	}
	for _, font := range ui.FontFamilies() {
		var name string
		if font.Name == sourceHanFamily {
			name = "思源黑体"
		} else {
			name = displayName(font.Name)
		}
		if _, exists := menu.familyByName[name]; exists {
			name += "（" + font.Name + "）"
		}
		menu.options = append(menu.options, name)
		menu.nameByFamily[font.Name], menu.familyByName[name] = name, font.Name
	}
	return menu
}
