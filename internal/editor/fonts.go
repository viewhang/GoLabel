package editor

import (
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
)

const (
	// sourceHanFamily 是思源黑体中国大陆版的字体族名
	sourceHanFamily = "Source Han Sans CN"
)

// defaultFontName 返回字体选择器中系统默认字体的名称
func defaultFontName() string { return i18n.Tr("font.system") }

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
var makeLabelFonts = makeFontMenu

// makeFontMenu 枚举系统字体并建立显示名和字体族名的对应关系
func makeFontMenu() fontMenu {
	defaultName := defaultFontName()
	menu := fontMenu{
		options:      []string{defaultName},
		nameByFamily: map[string]string{"": defaultName},
		familyByName: map[string]string{defaultName: ""},
	}
	for _, font := range ui.FontFamilies() {
		var name string
		if font.Name == sourceHanFamily {
			name = i18n.Tr("font.sourceHan")
		} else {
			name = platform.FontFamilyName(font.Name)
		}
		if _, exists := menu.familyByName[name]; exists {
			name += i18n.Trf("font.alias", font.Name)
		}
		menu.options = append(menu.options, name)
		menu.nameByFamily[font.Name], menu.familyByName[name] = name, font.Name
	}
	return menu
}

// fontPicker 显示整框可点击的字体搜索框并返回用户确认的选择
func fontPicker(c *ui.Context, selected *string, fonts fontMenu) bool {
	field := ui.Row(c).Grow(1).MinWidth(0).Height(34)
	query := ui.Local(field, "query", func() string { return "" })
	changed := false
	field.Children(func() {
		picker := ui.ComboboxBase(c.Key("input"), query)
		picker.Input.Grow(1).MinWidth(0).Height(34).Padding(0, 30, 0, 10).Radius(7).
			Border(1, colors.FieldBorder).Background(colors.FieldBg).FontSize(12).
			Placeholder(*selected).PlaceholderColor(colors.Text).Label(i18n.Tr("font.choose")).
			AutoComplete(ui.AutoCompleteList)
		ui.Icon(c.Key("arrow"), iconChevron).Size(13, 13).Rotate(90).TextColor(colors.Muted).
			Absolute().Right(10).Top(10).IgnoreInput(true)
		if !picker.Input.Focused() || !picker.Open() {
			*query = ""
		}
		picker.Popup(func(panel ui.Element) {
			panel.Margin(4, 0, 0, 0).Padding(4).Radius(8).
				Border(1, colors.FieldBorder).Background(colors.Panel)
			ui.Scroll(c.Key("options")).MaxHeight(260).Children(func() {
				filter := strings.ToLower(strings.TrimSpace(*query))
				matches := 0
				for _, name := range fonts.options {
					if !strings.Contains(strings.ToLower(name), filter) &&
						!strings.Contains(strings.ToLower(fonts.familyByName[name]), filter) {
						continue
					}
					matches++
					option := picker.Item(name).Padding(6, 8).Radius(5)
					if option.Highlighted() {
						option.Background(colors.Primary).TextColor(ui.RGB(255, 255, 255))
					} else if name == *selected {
						option.Background(colors.Selected)
					}
					option.Children(func() { ui.Text(c, name).FontSize(12).SingleLine() })
				}
				if matches == 0 {
					ui.Text(c, i18n.Tr("font.noMatches")).Padding(6, 8).FontSize(12).TextColor(colors.Muted)
				}
			})
		})
		if name, ok := picker.Chosen(); ok {
			changed = *selected != name
			*selected, *query = name, ""
		}
	})
	return changed
}
