package main

import (
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// saveImage 把测试器当前画面保存为 PNG，用于人工核对界面效果
func saveImage(t *testing.T, name string, tt *ui.Tester) {
	t.Helper()
	file, err := os.Create("/tmp/" + name + ".png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

// writeShot 渲染编辑器视图并保存为 PNG
func writeShot(t *testing.T, name string, e *editor, width, height int) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(e.view, width, height)
	tt.Frame()
	tt.Frame()
	saveImage(t, name, tt)
	return tt
}

// sampleLabel 返回用于预览的示例标签
func sampleLabel() *label {
	l := &label{Width: 50, Height: 30, DPI: 203}
	l.Items = []item{
		{Kind: kindText, X: 2.8, Y: 3, W: 27, H: 7, labelText: labelText{Value: "桂花乌龙", FontSize: 6, BlockAlign: ui.Center}},
		{Kind: kindText, X: 3, Y: 11, W: 20, H: 3, labelText: labelText{Value: "OSMANTHUS OOLONG", FontSize: 1.6}},
		{Kind: kindText, X: 3, Y: 15, W: 12, H: 3, labelText: labelText{Value: "净含量 50g", FontSize: 2}},
		{Kind: kindLine, X: 3, Y: 14, W: 30, H: 0.3},
		{Kind: kindTable, X: 30, Y: 18, W: 16, H: 9, Rows: 2, Columns: 2, Stroke: 0.3},
	}
	return l
}

// TestPreviewMain 渲染选中文字时的编辑器
func TestPreviewMain(t *testing.T) {
	e := &editor{label: sampleLabel(), selected: 0, zoom: 1}
	writeShot(t, "golabel-main", e, 1280, 800)

	e.selected = -1
	e.textEdit = nil
	writeShot(t, "golabel-label", e, 1280, 800)

	empty := &label{Width: 50, Height: 30, DPI: 203}
	writeShot(t, "golabel-empty", &editor{label: empty, selected: -1, zoom: 1}, 1280, 800)
}

// TestPreviewNewLabel 渲染新建标签对话框
func TestPreviewNewLabel(t *testing.T) {
	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := writeShot(t, "golabel-newlabel", e, 1280, 800)
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	saveImage(t, "golabel-newlabel", tt)

	if err := tt.Click("100 × 50 mm"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	saveImage(t, "golabel-newlabel-preset", tt)

	// 最小窗口下检查对话框不被裁切
	small := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	smallTT := writeShot(t, "golabel-newlabel-small", small, 980, 660)
	if err := smallTT.Click("新建"); err != nil {
		t.Fatal(err)
	}
	smallTT.Frame()
	smallTT.Frame()
	saveImage(t, "golabel-newlabel-small", smallTT)
}

// TestPreviewPrint 渲染打印预览对话框
func TestPreviewPrint(t *testing.T) {
	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := writeShot(t, "golabel-print", e, 1280, 800)
	if err := tt.Click("打印"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	saveImage(t, "golabel-print", tt)
}

// TestPreviewPanels 渲染表格属性和纸张设置弹层
func TestPreviewPanels(t *testing.T) {
	writeShot(t, "golabel-table", &editor{label: sampleLabel(), selected: 4, zoom: 1}, 1280, 800)

	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := writeShot(t, "golabel-paper", e, 1280, 800)
	if err := tt.Click("标签纸张设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	saveImage(t, "golabel-paper", tt)
}

// TestPreviewSettings 渲染设置通用页与模板库
func TestPreviewSettings(t *testing.T) {
	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := writeShot(t, "golabel-settings", e, 1280, 800)
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	saveImage(t, "golabel-settings", tt)

	if err := tt.Click("模板库"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	saveImage(t, "golabel-templates", tt)

	// 最小窗口下确认两个页面都不裁切
	small := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	small.settings = settingsState{open: true, page: settingsTemplates}
	writeShot(t, "golabel-templates-small", small, 980, 660)
}

// TestPreviewSettingsPanes 渲染设置中的打印页与关于页
func TestPreviewSettingsPanes(t *testing.T) {
	printed := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	printed.settings = settingsState{open: true, page: settingsPrint}
	writeShot(t, "golabel-settings-print", printed, 1280, 800)

	about := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	about.settings = settingsState{open: true, page: settingsAbout}
	writeShot(t, "golabel-settings-about", about, 1280, 800)
}

// TestPreviewCompact 渲染最小窗口下的编辑器顶栏
func TestPreviewCompact(t *testing.T) {
	writeShot(t, "golabel-main-small", &editor{label: sampleLabel(), selected: -1, zoom: 1}, 980, 660)
}
