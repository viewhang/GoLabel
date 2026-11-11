package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// templateSaveForm 保存命名对话框的输入和错误提示
type templateSaveForm struct {
	open    bool
	name    string
	message string
}

// userTemplateFile 保存模板名称和可复用的 GoLabel 标签数据
type userTemplateFile struct {
	Name    string          `json:"name"`
	Content json.RawMessage `json:"label"`
}

// templateDirectory 返回本机配置目录下的模板存放位置
func templateDirectory() (string, error) {
	path := settingsFile()
	if path == "" {
		return "", errors.New(tr("template.directoryError"))
	}
	return filepath.Join(filepath.Dir(path), "templates"), nil
}

// template 校验保存的标签内容并生成模板库条目
func (f userTemplateFile) template() (labelTemplate, error) {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return labelTemplate{}, errors.New(tr("template.nameRequired"))
	}
	l, err := readLabel(bytes.NewReader(f.Content))
	if err != nil {
		return labelTemplate{}, err
	}
	return labelTemplate{
		name: name, category: "custom", width: l.Width, height: l.Height, dpi: l.DPI, custom: l,
	}, nil
}

// loadUserTemplates 将本机已保存的标签加入模板库
func loadUserTemplates() {
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
		t, err := readUserTemplate(path)
		if err != nil {
			log.Printf("GoLabel: %s: %v", path, err)
			continue
		}
		labelTemplates = append(labelTemplates, t)
	}
}

// readUserTemplate 从磁盘读取并校验一个用户模板
func readUserTemplate(path string) (labelTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return labelTemplate{}, err
	}
	var file userTemplateFile
	if err := json.Unmarshal(data, &file); err != nil {
		return labelTemplate{}, err
	}
	return file.template()
}

// storeUserTemplate 把当前标签保存为独立的用户模板文件
func storeUserTemplate(name string, l *label) (labelTemplate, error) {
	name = strings.TrimSpace(name)
	for _, t := range labelTemplates {
		if t.custom != nil && strings.EqualFold(t.name, name) {
			return labelTemplate{}, errors.New(tr("template.duplicateName"))
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
	if e.document != nil && e.document.Path() != "" {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	e.saveForm = templateSaveForm{open: true, name: name}
}

// saveTemplateDialog 让用户为当前标签的模板副本命名
func (e *editor) saveTemplateDialog(c *ui.Context) {
	f := &e.saveForm
	ui.Modal(c, &f.open, func() {
		ui.Column(c.Key("dialog")).Width(340).Gap(12).Children(func() {
			ui.Text(c.Key("title"), tr("template.saveAs")).FontSize(18).FontWeight(650).TextColor(colors.Heading)
			ui.TextInput(c.Key("name"), &f.name).FillWidth().Height(34).AutoFocus().
				Label(tr("template.name")).Placeholder(tr("template.name"))
			if f.message != "" {
				ui.Text(c.Key("error"), f.message).FontSize(12).TextColor(colors.Danger)
			}
			ui.Row(c.Key("actions")).Gap(8).Justify(ui.End).Children(func() {
				plainAction(c.Key("cancel"), tr("new.cancel"), func() { f.open = false })
				accentAction(c.Key("save"), tr("template.confirmSave"), iconSave, strings.TrimSpace(f.name) != "", e.saveTemplate)
			})
		})
	})
}

// saveTemplate 保存标签副本并在我的模板中选中新条目
func (e *editor) saveTemplate() {
	t, err := storeUserTemplate(e.saveForm.name, e.label)
	if err != nil {
		e.saveForm.message = err.Error()
		return
	}
	labelTemplates = append(labelTemplates, t)
	e.saveForm.open = false
	e.status = tr("template.saved")
	e.openSettings(settingsTemplates)
	e.settings.query = ""
	e.settings.category = slices.Index(templateCategories[:], "custom")
	e.settings.template = len(labelTemplates) - 1
}
