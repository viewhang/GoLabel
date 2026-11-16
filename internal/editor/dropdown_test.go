package editor

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestFontDropdownOpensAcrossField 验证文字和单元格的字体框各处均可展开并应用字体
func TestFontDropdownOpensAcrossField(t *testing.T) {
	restoreFonts, restorePrefs := makeLabelFonts, prefs
	t.Cleanup(func() {
		makeLabelFonts = restoreFonts
		prefs = restorePrefs
		resetFontMenu()
		savePreferences()
	})
	makeLabelFonts = func() fontMenu {
		return fontMenu{
			options:      []string{defaultFontName(), "测试字体"},
			nameByFamily: map[string]string{"": defaultFontName(), "Arial": "测试字体"},
			familyByName: map[string]string{defaultFontName(): "", "测试字体": "Arial"},
		}
	}
	resetFontMenu()
	for _, kind := range []itemKind{kindText, kindTable} {
		for _, position := range []string{"左侧内边距", "当前字体", "下拉箭头"} {
			t.Run(string(kind)+"/"+position, func(t *testing.T) {
				l := newLabel()
				l.Items = []item{{Kind: kind, X: 2, Y: 2, W: 25, H: 10, Rows: 1, Columns: 1, Stroke: 0.3,
					labelText: labelText{Value: "字体测试", FontSize: 4}}}
				e := &editor{label: l, selected: 0, zoom: 1}
				if kind == kindTable {
					l.Items[0].setCell(0, 0, labelText{Value: "字体测试", FontSize: 4})
					e.cells = &tableSelection{}
				}
				tt := ui.NewTester(e.view, 1280, 900)
				tt.Frame()
				field, ok := tt.Find("选择字体")
				if !ok {
					t.Fatal("找不到字体选择框")
				}
				x := field.X + 2
				if position == "当前字体" {
					x = field.X + 30
				} else if position == "下拉箭头" {
					x = field.X + field.W - 10
				}
				tt.ClickAt(x, field.Y+field.H/2)
				if !tt.HasText("测试字体") {
					t.Fatal("点击字体框没有展开字体列表")
				}
				if err := tt.Click("测试字体"); err != nil {
					t.Fatal(err)
				}
				text := l.Items[0].labelText
				if kind == kindTable {
					text = l.Items[0].cellValue(0, 0)
				}
				if font := newTextEdit(text).doc.Formatting().Attributes.Font; font != "Arial" {
					t.Fatalf("所选字体未写入文字格式：%q", font)
				}
			})
		}
	}
}

// TestFontDropdownSearch 验证中英文搜索、键盘确认、无匹配提示和取消后重新展开
func TestFontDropdownSearch(t *testing.T) {
	restore := prefs
	t.Cleanup(func() {
		prefs = restore
		savePreferences()
	})
	fonts := fontMenu{
		options: []string{defaultFontName(), "思源黑体", "阿里巴巴普惠体"},
		familyByName: map[string]string{
			defaultFontName(): "", "思源黑体": sourceHanFamily, "阿里巴巴普惠体": "Alibaba PuHuiTi",
		},
	}
	selected, changes := defaultFontName(), 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Width(240).Children(func() {
			if fontPicker(c, &selected, fonts) {
				changes++
			}
		})
	}, 300, 400)
	for _, search := range []struct{ query, wanted, hidden string }{
		{"思源", "思源黑体", "阿里巴巴普惠体"},
		{"ALIBABA", "阿里巴巴普惠体", "思源黑体"},
	} {
		if err := tt.Click("选择字体"); err != nil {
			t.Fatal(err)
		}
		tt.Type(search.query)
		if !tt.HasText(search.wanted) || tt.HasText(search.hidden) {
			t.Fatalf("搜索 %q 的结果不正确：%v", search.query, tt.Texts())
		}
		tt.Key(0, ui.KeyEnter)
		if selected != search.wanted {
			t.Fatalf("回车未应用搜索结果：%q", selected)
		}
	}
	if changes != 2 {
		t.Fatalf("字体变更次数不正确：%d", changes)
	}
	if err := tt.Click("选择字体"); err != nil {
		t.Fatal(err)
	}
	tt.Type("不存在的字体")
	if !tt.HasText("未找到字体") || selected != "阿里巴巴普惠体" {
		t.Fatal("无匹配字体时未提示或修改了原字体")
	}
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("选择字体"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("思源黑体") || tt.HasText("未找到字体") {
		t.Fatal("取消搜索后重新打开未恢复完整字体列表")
	}
}

// TestFontDropdownHistory 验证名称排序、历史置顶去重、重启恢复和切换语言
func TestFontDropdownHistory(t *testing.T) {
	t.Cleanup(savePreferences)
	resetLanguage(t)
	setAppLocale(string(languageZhHans))
	prefs.RecentFonts = nil
	fonts := fontMenu{
		options: []string{"微软雅黑", "宋体", "阿里巴巴普惠体", "仿宋"},
		familyByName: map[string]string{
			"微软雅黑": "Microsoft YaHei", "宋体": "SimSun", "阿里巴巴普惠体": "Alibaba PuHuiTi", "仿宋": "FangSong",
		},
	}
	if got := fonts.orderedOptions(); !slices.Equal(got, []string{"阿里巴巴普惠体", "仿宋", "宋体", "微软雅黑"}) {
		t.Fatalf("字体未按拼音排序：%v", got)
	}
	selected := "宋体"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Width(240).Children(func() { fontPicker(c, &selected, fonts) })
	}, 300, 400)
	for _, name := range []string{"微软雅黑", "宋体", "微软雅黑"} {
		if err := tt.Click("选择字体"); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(name); err != nil {
			t.Fatal(err)
		}
	}
	wantHistory := []string{"Microsoft YaHei", "SimSun"}
	if !slices.Equal(prefs.RecentFonts, wantHistory) {
		t.Fatalf("字体历史未按最近使用去重：%v", prefs.RecentFonts)
	}
	prefs = defaultPreferences()
	loadPreferences()
	if !slices.Equal(prefs.RecentFonts, wantHistory) {
		t.Fatalf("字体历史未在重启后恢复：%v", prefs.RecentFonts)
	}
	if got := fonts.orderedOptions(); !slices.Equal(got, []string{"微软雅黑", "宋体", "阿里巴巴普惠体", "仿宋"}) {
		t.Fatalf("字体历史未置顶或其余名称排序有误：%v", got)
	}
	if err := tt.Click("选择字体"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if selected != "微软雅黑" || !slices.Equal(prefs.RecentFonts, wantHistory) {
		t.Fatal("下拉框首项不是最近使用的字体或重复记录了当前字体")
	}
	setAppLocale(string(languageEnglish))
	english := fontMenu{
		options:      []string{"SimSun", "Microsoft YaHei", "FangSong", "Alibaba PuHuiTi"},
		familyByName: map[string]string{},
	}
	for _, name := range english.options {
		english.familyByName[name] = name
	}
	if got := english.orderedOptions(); !slices.Equal(got, []string{"Microsoft YaHei", "SimSun", "Alibaba PuHuiTi", "FangSong"}) {
		t.Fatalf("切换语言后历史字体丢失或英文名称未排序：%v", got)
	}
}

// TestSettingsDropdownsOpenFromValue 验证设置中的尺寸和分辨率点击当前值即可选择
func TestSettingsDropdownsOpenFromValue(t *testing.T) {
	resetLanguage(t)
	prefs = defaultPreferences()
	prefs.DefaultWidth, prefs.DefaultHeight, prefs.DefaultDPI = 40, 30, 203
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(700).Children(func() {
			e.defaultSizeRow(c.Key("size"))
			e.defaultDPIRow(c.Key("dpi"))
		})
	}, 750, 200)
	tt.Frame()
	for _, choice := range []struct{ current, next string }{
		{"40 × 30 mm", "60 × 40 mm"},
		{"203 DPI", "300 DPI"},
	} {
		if err := tt.Click(choice.current); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(choice.next); err != nil {
			t.Fatal(err)
		}
	}
	if prefs.DefaultWidth != 60 || prefs.DefaultHeight != 40 || prefs.DefaultDPI != 300 {
		t.Fatal("下拉选择未更新默认尺寸或分辨率")
	}
}
