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

// TestUserTemplateManagement 验证同名模板覆盖、重载和删除操作保持文件与模板库一致
func TestUserTemplateManagement(t *testing.T) {
	originalTemplates, originalSettings := labelTemplates, settingsFile
	dir := t.TempDir()
	settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	labelTemplates = slices.Clone(originalTemplates)
	t.Cleanup(func() { labelTemplates, settingsFile = originalTemplates, originalSettings })
	l := newLabel()
	l.Items = []item{tplText(2, 2, 20, 6, 4, "保存前", ui.Start)}
	e := &editor{label: l, selected: -1, zoom: 1}
	e.saveForm.name = "商品模板 A"
	e.saveTemplate()
	index := len(labelTemplates) - 1
	if e.saveForm.message != "" || len(labelTemplates) != len(originalTemplates)+1 {
		t.Fatalf("首次保存失败：%s", e.saveForm.message)
	}
	first := labelTemplates[index]
	preview := func(template labelTemplate) *image.RGBA {
		return ui.Render(func(c *ui.Context) {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { paintTemplateThumb(p, r, template) })
		}, 320, 240, 1)
	}

	labelTemplates = slices.Clone(originalTemplates)
	loadUserTemplates(nil)
	l.Items[0].Value = "覆盖后的文字"
	l.DPI = 300
	e.saveForm.name = " 商品模板 a "
	e.saveTemplate()
	updated := labelTemplates[index]
	if e.saveForm.message != "" || len(labelTemplates) != len(originalTemplates)+1 || updated.path != first.path {
		t.Fatalf("同名模板未覆盖原文件：%s", e.saveForm.message)
	}
	files, err := filepath.Glob(filepath.Join(dir, "templates", "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("覆盖后产生了多份文件：%v，%v", files, err)
	}
	stored, err := readUserTemplate(first.path, nil)
	if err != nil || stored.custom.Items[0].Value != "覆盖后的文字" || stored.dpi != 300 ||
		stored.name != "商品模板 a" || bytes.Equal(preview(first).Pix, preview(updated).Pix) {
		t.Fatalf("覆盖后没有保存新的内容、名称、分辨率和缩略图：%v", err)
	}
	if !bytes.Equal(preview(stored).Pix, preview(updated).Pix) {
		t.Fatal("覆盖后的缩略图没有正确持久化")
	}
	l.Items[0].Value = "文档继续编辑"
	if updated.custom.Items[0].Value != "覆盖后的文字" {
		t.Fatal("覆盖后的模板与文档共用了编辑数据")
	}
	e.saveForm.name = "另一个模板"
	e.saveTemplate()
	tt := ui.NewTester(e.view, 1280, 800)
	if _, ok := tt.AccessByRole(ui.RoleButton, "删除模板"); ok {
		t.Fatal("未悬停时显示了删除入口")
	}
	hoverTemplate := func(name string) {
		t.Helper()
		card, ok := tt.AccessByRole(ui.RoleButton, name)
		if !ok {
			t.Fatal("找不到用户模板卡片")
		}
		tt.Move(card.Bounds.X+card.Bounds.W/2, card.Bounds.Y+20)
		button, ok := tt.AccessByRole(ui.RoleButton, "删除模板")
		if !ok || button.Bounds.X < card.Bounds.X+card.Bounds.W-32 || button.Bounds.Y > card.Bounds.Y+8 {
			saveImage(t, "golabel-template-hover", tt)
			t.Fatalf("悬停后卡片右上角没有显示删除入口：卡片 %+v，按钮 %+v，存在 %v", card.Bounds, button.Bounds, ok)
		}
	}
	hoverTemplate("商品模板 a")
	if e.settings.template == index {
		t.Fatal("悬停未选中模板时改变了选中状态")
	}
	if err := tt.Click("删除模板"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if len(labelTemplates) != len(originalTemplates)+2 {
		t.Fatal("取消删除仍然修改了模板库")
	}
	hoverTemplate("商品模板 a")
	if err := tt.Click("删除模板"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if len(labelTemplates) != len(originalTemplates)+1 || labelTemplates[len(labelTemplates)-1].name != "另一个模板" {
		t.Fatal("删除操作未移除选中的模板或移除了其他模板")
	}
	if _, err := os.Stat(first.path); !os.IsNotExist(err) {
		t.Fatalf("删除模板后文件仍存在：%v", err)
	}
	if e.label != l || e.label.Items[0].Value != "文档继续编辑" {
		t.Fatal("删除模板影响了当前编辑文档")
	}
	hoverTemplate("另一个模板")
	saveImage(t, "golabel-template-management", tt)
	if err := tt.Click("删除模板"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	_, hasUse := tt.AccessByRole(ui.RoleButton, "使用模板")
	_, hasDelete := tt.AccessByRole(ui.RoleButton, "删除模板")
	if !tt.HasText("没有匹配的模板") || hasUse || hasDelete {
		t.Fatalf("删除最后一个用户模板后仍显示失效的预览或操作：模板数 %d，状态 %+v", len(labelTemplates), e.settings)
	}
	labelTemplates = slices.Clone(originalTemplates)
	loadUserTemplates(nil)
	if len(labelTemplates) != len(originalTemplates) {
		t.Fatal("已删除的模板在重新加载后又出现")
	}
	e.settings.category = 0
	tt.Frame()
	builtin, ok := tt.AccessByRole(ui.RoleButton, originalTemplates[0].title())
	if !ok {
		t.Fatal("找不到内置模板卡片")
	}
	tt.Move(builtin.Bounds.X+builtin.Bounds.W/2, builtin.Bounds.Y+20)
	if _, ok := tt.AccessByRole(ui.RoleButton, "删除模板"); ok {
		t.Fatal("内置模板出现了删除入口")
	}
}
