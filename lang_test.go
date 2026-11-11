package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// resetLanguage 在测试结束后恢复语言和偏好，避免影响其他用例
func resetLanguage(t *testing.T) {
	t.Helper()
	restorePrefs, restoreLang := prefs, appLocale()
	t.Cleanup(func() {
		prefs = restorePrefs
		setAppLocale(restoreLang)
		applyTheme()
	})
}

// TestApplyLanguage 验证跟随系统和无效偏好都交给 MyGo 解析
func TestApplyLanguage(t *testing.T) {
	resetLanguage(t)
	setLocale := setAppLocale
	t.Cleanup(func() { setAppLocale = setLocale })
	setAppLocale = func(tag string) {
		if tag != "" {
			t.Fatalf("跟随系统时应清除语言覆盖，实际收到 %q", tag)
		}
		setLocale(string(languageEnglish))
	}
	for _, saved := range []string{string(languageSystem), "klingon"} {
		prefs.Language = saved
		applyLanguage()
		if got := tr("header.save"); got != "Save" {
			t.Fatalf("偏好 %q 没有使用 MyGo 解析的英文：%q", saved, got)
		}
	}
}

// TestLanguageSwitch 验证切换到英文后顶栏、侧栏和属性栏都显示英文
func TestLanguageSwitch(t *testing.T) {
	resetLanguage(t)
	useLanguage(languageEnglish)

	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	for _, want := range []string{"New", "Open", "Save", "Templates", "Settings", "Export PDF",
		"Print", "Add element", "Layers", "Size", "Resolution", "Elements on canvas"} {
		if !tt.HasText(want) {
			t.Fatalf("英文界面缺少 %q：%v", want, tt.Texts())
		}
	}
	for _, chinese := range []string{"新建", "添加元素", "图层", "画布中的元素"} {
		if tt.HasText(chinese) {
			t.Fatalf("切换到英文后仍显示中文 %q：%v", chinese, tt.Texts())
		}
	}
}

// TestLanguageCardInSettings 验证点击当前语言文字可展开选项，切换后保留标签栏的保存状态
func TestLanguageCardInSettings(t *testing.T) {
	resetLanguage(t)
	prefs.Language = string(languageZhHans)
	applyLanguage()

	e := &editor{label: sampleLabel(), tabs: &documentTabs{}, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if prefs.Language != string(languageEnglish) {
		t.Fatalf("语言没有保存为英文：%q", prefs.Language)
	}
	if got := appLocale(); got != string(languageEnglish) {
		t.Fatalf("切换后界面语言不是英文：%q", got)
	}
	for _, want := range []string{"General", "Appearance", "Language", "Light", "Dark"} {
		if !tt.HasText(want) {
			t.Fatalf("英文设置页缺少 %q：%v", want, tt.Texts())
		}
	}
	tt.Key(0, ui.KeyEscape)
	if !tt.HasText("Draft") || tt.HasText("Switched to English") {
		t.Fatalf("切换语言后标签栏未保留草稿状态：%v", tt.Texts())
	}

	// 再切回跟随系统。「跟随系统」和主题卡片同名，这里直接调用切换函数
	useLanguage(languageSystem)
	if prefs.Language != string(languageSystem) {
		t.Fatalf("没有切回跟随系统：%q", prefs.Language)
	}
}

// TestLanguagePreferenceRoundTrip 验证语言会写入配置并在下次启动恢复
func TestLanguagePreferenceRoundTrip(t *testing.T) {
	resetLanguage(t)
	prefs = defaultPreferences()
	prefs.Language = string(languageEnglish)
	savePreferences()
	prefs = defaultPreferences()
	loadPreferences()
	if prefs.Language != string(languageEnglish) {
		t.Fatalf("语言没有恢复：%q", prefs.Language)
	}

	// 不认识的取值应回落到跟随系统
	broken := `{"defaultWidthMm":50,"defaultHeightMm":30,"defaultDpi":203,"language":"klingon"}`
	if err := os.WriteFile(settingsFile(), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	prefs = defaultPreferences()
	loadPreferences()
	if prefs.Language != string(languageSystem) {
		t.Fatalf("无效语言没有回落到跟随系统：%q", prefs.Language)
	}
}

// TestTemplateContentFollowsLanguage 验证模板的示例内容跟着界面语言切换
func TestTemplateContentFollowsLanguage(t *testing.T) {
	resetLanguage(t)
	for i, template := range labelTemplates {
		if len(template.enItems) == 0 {
			t.Fatalf("第 %d 个模板缺少英文示例内容", i+1)
		}
	}
	setAppLocale(string(languageZhHans))
	chinese := labelTemplates[0].label()
	setAppLocale(string(languageEnglish))
	english := labelTemplates[0].label()
	if chinese.Items[1].Value == english.Items[1].Value {
		t.Fatalf("中英文模板内容相同：%q", chinese.Items[1].Value)
	}
	if english.Items[1].Value != "Oolong Tea" {
		t.Fatalf("英文模板没有取到英文内容：%q", english.Items[1].Value)
	}
	if labelTemplates[0].title() != "Product label" {
		t.Fatalf("英文下的模板名称不对：%q", labelTemplates[0].title())
	}
	if labelTemplates[1].title() != "Food label" {
		t.Fatalf("英文下的模板名称不对：%q", labelTemplates[1].title())
	}
}

// TestMessagesComplete 验证代码里用到的文案 key 都在文案表里，且两种语言都不为空
func TestMessagesComplete(t *testing.T) {
	used := make(map[string]bool)
	sources := new(strings.Builder)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`trf?\("([^"]+)"`)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sources.Write(data)
		for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
			used[match[1]] = true
		}
	}
	for key := range used {
		// 以点结尾的 key 是拼接用的前缀，不是完整文案
		if strings.HasSuffix(key, ".") {
			continue
		}
		if _, ok := messages[key]; !ok {
			t.Errorf("文案表缺少 key %q", key)
		}
	}
	source := sources.String()
	for key := range messages {
		if used[key] {
			continue
		}
		// 模板分类的 key 是拼出来的，按前缀判断
		if strings.HasPrefix(key, "template.category.") && strings.Contains(source, `"template.category.`) {
			continue
		}
		if !strings.Contains(source, `"`+key+`"`) {
			t.Errorf("文案 %q 没有被使用", key)
		}
	}
	for key, message := range messages {
		if strings.TrimSpace(message.zh) == "" || strings.TrimSpace(message.en) == "" {
			t.Errorf("文案 %q 缺少中文或英文", key)
		}
	}
}

// TestPreviewEnglish 渲染英文界面，便于人工核对排版
func TestPreviewEnglish(t *testing.T) {
	resetLanguage(t)
	useLanguage(languageEnglish)

	writeShot(t, "golabel-en-main", &editor{label: sampleLabel(), selected: 0, zoom: 1}, 1280, 800)

	empty := &label{Width: 50, Height: 30, DPI: 203}
	writeShot(t, "golabel-en-empty", &editor{label: empty, selected: -1, zoom: 1}, 1280, 800)

	settings := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	settings.settings = settingsState{open: true, page: settingsGeneral}
	writeShot(t, "golabel-en-settings", settings, 1280, 800)

	templates := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	templates.settings = settingsState{open: true, page: settingsTemplates}
	writeShot(t, "golabel-en-templates", templates, 1280, 800)

	creator := &editor{label: sampleLabel(), selected: -1, zoom: 1, form: newLabelForm{
		open: true, width: 50, height: 30, dpi: 203, name: "Untitled label",
	}}
	writeShot(t, "golabel-en-newlabel", creator, 1280, 800)

	printer := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	printer.preview = printPreview{open: true}
	writeShot(t, "golabel-en-print", printer, 1280, 800)

	about := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	about.settings = settingsState{open: true, page: settingsAbout}
	writeShot(t, "golabel-en-about", about, 1280, 800)
}
