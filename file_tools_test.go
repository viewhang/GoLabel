package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestRecentFilesMenu 验证打开旁的菜单会列出仍存在的最近文件
func TestRecentFilesMenu(t *testing.T) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		t.Fatal(err)
	}
	labelPath := filepath.Join(dir, "商品.golabel")
	var labelData bytes.Buffer
	if err := writeLabel(&labelData, newLabel()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(labelPath, labelData.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal([]string{filepath.Join(dir, "已删除.golabel"), labelPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "recent-documents.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	if err := tt.Click("最近文件"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Menu(); !slices.Equal(got, []string{"商品.golabel"}) {
		t.Fatalf("最近文件菜单显示了 %v", got)
	}
}

// TestSaveTemplateRoundTrip 验证模板命名、持久化和包含图片的独立副本
func TestSaveTemplateRoundTrip(t *testing.T) {
	originalTemplates := labelTemplates
	originalSettingsFile := settingsFile
	dir := t.TempDir()
	settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	t.Cleanup(func() {
		labelTemplates = originalTemplates
		settingsFile = originalSettingsFile
	})
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	l := newLabel()
	l.Items = []item{
		tplText(2, 2, 20, 6, 4, "原始文字", ui.Start),
		{Kind: kindImage, X: 2, Y: 10, W: 10, H: 10, Data: picture.Bytes()},
	}
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	if err := tt.Click("保存选项"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("保存为模板"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("模板名称"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("商品模板 A")
	if err := tt.Click("保存模板"); err != nil {
		t.Fatal(err)
	}
	if e.saveForm.open || !e.settings.open || !tt.HasText("商品模板 A") {
		t.Fatalf("模板保存后未显示在库中：%q", e.saveForm.message)
	}

	labelTemplates = originalTemplates
	loadUserTemplates()
	if len(labelTemplates) != len(originalTemplates)+1 {
		t.Fatal("重新加载时没有读到已保存的模板")
	}
	l.Items[0].Value = "原标签后来改了"
	tt.Frame()
	if err := tt.Click("使用模板"); err != nil {
		t.Fatal(err)
	}
	if e.label == l || e.label.Items[0].Value != "原始文字" || e.label.Items[1].bitmap == nil {
		t.Fatal("使用模板后未生成包含原始内容和图片的独立标签")
	}
}
