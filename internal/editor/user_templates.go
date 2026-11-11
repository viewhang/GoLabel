package editor

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
	"github.com/pengh/GoLabel/internal/platform"
)

// templateSaveForm 保存命名对话框的输入和错误提示
type templateSaveForm struct {
	open    bool
	name    string
	message string
}

// userTemplateFile 保存模板名称、标签数据和实际内容的 PNG 缩略图
type userTemplateFile struct {
	Name      string          `json:"name"`
	Content   json.RawMessage `json:"label"`
	Thumbnail []byte          `json:"thumbnail,omitempty"`
}

// templateDirectory 返回本机配置目录下的模板存放位置
func templateDirectory() (string, error) {
	path := settingsFile()
	if path == "" {
		return "", errors.New(i18n.Tr("template.directoryError"))
	}
	return filepath.Join(filepath.Dir(path), "templates"), nil
}

// template 校验保存的标签内容并生成模板库条目
func (f userTemplateFile) template() (labelTemplate, error) {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return labelTemplate{}, errors.New(i18n.Tr("template.nameRequired"))
	}
	l, err := readLabel(bytes.NewReader(f.Content))
	if err != nil {
		return labelTemplate{}, err
	}
	t := labelTemplate{
		name: name, category: "custom", width: l.Width, height: l.Height, dpi: l.DPI, custom: l,
	}
	if len(f.Thumbnail) > 0 {
		t.thumbnail, err = decodeImage(f.Thumbnail)
		if err != nil {
			return labelTemplate{}, err
		}
	}
	return t, nil
}

// loadUserTemplates 将本机已保存的标签加入模板库
func loadUserTemplates(graphics *platform.Graphics) {
	dir, err := templateDirectory()
	if err != nil {
		log.Print(err)
		return
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		log.Print(err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		t, err := readUserTemplate(path, graphics)
		if err != nil {
			log.Printf("GoLabel: %s: %v", path, err)
			continue
		}
		labelTemplates = append(labelTemplates, t)
	}
}

// readUserTemplate 读取用户模板并为旧模板生成实际内容预览
func readUserTemplate(path string, graphics *platform.Graphics) (labelTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return labelTemplate{}, err
	}
	var file userTemplateFile
	if err := json.Unmarshal(data, &file); err != nil {
		return labelTemplate{}, err
	}
	t, err := file.template()
	if err != nil {
		return labelTemplate{}, err
	}
	if t.thumbnail == nil {
		for i := range t.custom.Items {
			element := &t.custom.Items[i]
			if graphics != nil && (element.Kind == kindBarcode || element.Kind == kindQR) {
				element.prepareSymbol(graphics)
			}
		}
		t.thumbnail = ui.NewBitmap(renderTemplateThumbnail(t.custom))
	}
	return t, nil
}

// renderTemplateThumbnail 按标签比例使用画布绘制逻辑生成缩略图
func renderTemplateThumbnail(l *label) *image.RGBA {
	scale := float64(templateThumbnailSize) / max(l.Width, l.Height)
	width, height := int(math.Round(l.Width*scale)), int(math.Round(l.Height*scale))
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) { l.paint(p, r, false) })
	}, width, height, 1)
}

// storeUserTemplate 将当前标签和即时生成的缩略图保存为同一个模板文件
func storeUserTemplate(name string, l *label) (labelTemplate, error) {
	name = strings.TrimSpace(name)
	for _, t := range labelTemplates {
		if t.custom != nil && strings.EqualFold(t.name, name) {
			return labelTemplate{}, errors.New(i18n.Tr("template.duplicateName"))
		}
	}
	content, err := json.Marshal(l)
	if err != nil {
		return labelTemplate{}, err
	}
	file := userTemplateFile{Name: name, Content: content}
	t, err := file.template()
	if err != nil {
		return labelTemplate{}, err
	}
	thumbnail := renderTemplateThumbnail(l)
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, thumbnail); err != nil {
		return labelTemplate{}, err
	}
	file.Thumbnail = imageData.Bytes()
	t.thumbnail = ui.NewBitmap(thumbnail)
	dir, err := templateDirectory()
	if err != nil {
		return labelTemplate{}, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return labelTemplate{}, err
	}
	tmp, err := os.CreateTemp(dir, "template-*")
	if err != nil {
		return labelTemplate{}, err
	}
	defer os.Remove(tmp.Name())
	writeErr := json.NewEncoder(tmp).Encode(file)
	if err := errors.Join(writeErr, tmp.Close()); err != nil {
		return labelTemplate{}, err
	}
	if err := os.Rename(tmp.Name(), tmp.Name()+".json"); err != nil {
		return labelTemplate{}, err
	}
	return t, nil
}

// openSaveTemplate 打开命名对话框并预填当前标签名称
func (e *editor) openSaveTemplate() {
	name := e.documentName()
	if e.document != nil && e.document.path != "" {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	e.saveForm = templateSaveForm{open: true, name: name}
}

// saveTemplateDialog 让用户为当前标签的模板副本命名
func (e *editor) saveTemplateDialog(c *ui.Context) {
	f := &e.saveForm
	ui.Modal(c, &f.open, func() {
		ui.Column(c.Key("dialog")).Width(340).Gap(12).Children(func() {
			ui.Text(c.Key("title"), i18n.Tr("template.saveAs")).FontSize(18).FontWeight(650).TextColor(colors.Heading)
			ui.TextInput(c.Key("name"), &f.name).FillWidth().Height(34).AutoFocus().
				Label(i18n.Tr("template.name")).Placeholder(i18n.Tr("template.name"))
			if f.message != "" {
				ui.Text(c.Key("error"), f.message).FontSize(12).TextColor(colors.Danger)
			}
			ui.Row(c.Key("actions")).Gap(8).Justify(ui.End).Children(func() {
				plainAction(c.Key("cancel"), i18n.Tr("new.cancel"), func() { f.open = false })
				accentAction(c.Key("save"), i18n.Tr("template.confirmSave"), iconSave, strings.TrimSpace(f.name) != "", e.saveTemplate)
			})
		})
	})
}

// saveTemplate 写回正在编辑的文字并保存带缩略图的模板副本
func (e *editor) saveTemplate() {
	e.flushTextEdit()
	t, err := storeUserTemplate(e.saveForm.name, e.label)
	if err != nil {
		e.saveForm.message = err.Error()
		return
	}
	labelTemplates = append(labelTemplates, t)
	e.saveForm.open = false
	e.status = i18n.Tr("template.saved")
	e.openSettings(settingsTemplates)
	e.settings.query = ""
	e.settings.category = slices.Index(templateCategories[:], "custom")
	e.settings.template = len(labelTemplates) - 1
}
