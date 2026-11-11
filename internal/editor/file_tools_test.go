package editor

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
	"github.com/egoist/mygo/ui/richtext"
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

// TestUserTemplateThumbnail 验证保存时写回富文本并持久化实际缩略图且兼容旧模板
func TestUserTemplateThumbnail(t *testing.T) {
	originalTemplates, originalSettings := labelTemplates, settingsFile
	dir := t.TempDir()
	settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	t.Cleanup(func() {
		labelTemplates, settingsFile = originalTemplates, originalSettings
	})
	l := &label{Width: 40, Height: 30, DPI: 203, Items: []item{
		{Kind: kindText, X: 1, Y: 1, W: 38, H: 12, labelText: labelText{Value: "原始文字", FontSize: 7}},
	}}
	e := &editor{label: l, selected: 0, zoom: 1, textEdit: newTextEdit(l.Items[0].labelText)}
	length := e.textEdit.doc.Text().Len()
	e.textEdit.doc.Edit(func(edit *richtext.Edit) {
		edit.ReplaceString(0, length, "1.5 单被套", richtext.Attributes{Weight: 700})
	})
	e.saveForm.name = "富文本缩略图"
	e.saveTemplate()
	if e.saveForm.message != "" {
		t.Fatal(e.saveForm.message)
	}
	saveImage(t, "golabel-custom-template-thumbnail", ui.NewTester(e.view, 1280, 800))
	template := labelTemplates[len(labelTemplates)-1]
	if template.thumbnail == nil || template.custom.Items[0].RichText == nil ||
		template.custom.Items[0].Value != "" || template.custom.Items[0].textContent().PlainText() != "1.5 单被套" {
		t.Fatal("保存时没有写回富文本或生成实际缩略图")
	}
	files, err := filepath.Glob(filepath.Join(dir, "templates", "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("模板文件数量不正确：%v，%v", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var saved userTemplateFile
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	thumbnail, err := png.Decode(bytes.NewReader(saved.Thumbnail))
	if err != nil {
		t.Fatalf("没有持久化有效 PNG 缩略图：%v", err)
	}
	if thumbnail.Bounds().Dx() != 640 || thumbnail.Bounds().Dy() != 480 {
		t.Fatalf("缩略图没有保持标签宽高比：%v", thumbnail.Bounds())
	}
	ink := 0
	for y := 0; y < thumbnail.Bounds().Dy(); y++ {
		for x := 0; x < thumbnail.Bounds().Dx(); x++ {
			r, g, b, _ := thumbnail.At(x, y).RGBA()
			if r+g+b < 3*0x4000 {
				ink++
			}
		}
	}
	if ink < 100 {
		t.Fatal("富文本缩略图仍然是空白")
	}
	loaded, err := readUserTemplate(files[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	render := func(template labelTemplate) *image.RGBA {
		return ui.Render(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paintTemplateThumb(p, r, template) })
		}, 320, 240, 1)
	}
	first := render(template)
	if !bytes.Equal(first.Pix, render(loaded).Pix) {
		t.Fatal("重新加载后没有使用相同的缩略图")
	}
	saved.Thumbnail = nil
	legacy, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(files[0], legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err = readUserTemplate(files[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.thumbnail == nil || !bytes.Equal(first.Pix, render(loaded).Pix) {
		t.Fatal("已有模板没有从富文本内容重建正确的预览")
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
	loadUserTemplates(nil)
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
