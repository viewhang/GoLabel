package main

import (
	"sync"

	"github.com/egoist/mygo/ui"
)

const (
	// sourceHanFamily 是思源黑体中国大陆版的字体族名
	sourceHanFamily = "Source Han Sans CN"
)

// defaultFontName 返回字体选择器中系统默认字体的名称
func defaultFontName() string { return tr("font.system") }

// fontMenu 保存字体菜单显示名称与模板字体族名的对应关系
type fontMenu struct {
	options      []string
	nameByFamily map[string]string
	familyByName map[string]string
}

// labelFonts 缓存系统字体菜单
var labelFonts = sync.OnceValue(func() fontMenu { return makeLabelFonts() })

// resetFontMenu 丢弃已缓存的字体菜单，切换界面语言后重新枚举
func resetFontMenu() {
	labelFonts = sync.OnceValue(func() fontMenu { return makeLabelFonts() })
}

// makeLabelFonts 枚举系统字体；测试中替换为不访问原生字体服务的版本
var makeLabelFonts = systemFontMenu

// makeFontMenu 枚举系统字体并建立显示名和字体族名的对应关系
func makeFontMenu(displayName func(string) string) fontMenu {
	defaultName := defaultFontName()
	menu := fontMenu{
		options:      []string{defaultName},
		nameByFamily: map[string]string{"": defaultName},
		familyByName: map[string]string{defaultName: ""},
	}
	for _, font := range ui.FontFamilies() {
		var name string
		if font.Name == sourceHanFamily {
			name = tr("font.sourceHan")
		} else {
			name = displayName(font.Name)
		}
		if _, exists := menu.familyByName[name]; exists {
			name += trf("font.alias", font.Name)
		}
		menu.options = append(menu.options, name)
		menu.nameByFamily[font.Name], menu.familyByName[name] = name, font.Name
	}
	return menu
}
