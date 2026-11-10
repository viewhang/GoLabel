package main

import (
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestToolbarAddsText 验证点击左侧工具卡片会添加并选中文字元素
func TestToolbarAddsText(t *testing.T) {
	l := newLabel()
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("文字"); err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 1 || l.Items[0].Kind != kindText || e.selected != 0 {
		t.Fatalf("点击工具栏未添加文字：%d 个元素，选中 %d", len(l.Items), e.selected)
	}
	if e.textEdit == nil {
		t.Fatal("添加文字后没有创建编辑会话")
	}
}

// TestLayerRowSelects 验证点击图层行会切换选中元素
func TestLayerRowSelects(t *testing.T) {
	l := newLabel()
	l.Items = []item{
		{Kind: kindText, X: 2, Y: 2, W: 20, H: 6, labelText: labelText{Value: "标题", FontSize: 4}},
		{Kind: kindLine, X: 2, Y: 10, W: 20, H: 0.3},
	}
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("分隔线"); err != nil {
		t.Fatal(err)
	}
	if e.selected != 1 {
		t.Fatalf("点击图层未选中分隔线：选中 %d", e.selected)
	}
}

// TestFloatingToolbarKeepsSelection 验证浮动工具栏的按钮不会开始拖动元素
func TestFloatingToolbarKeepsSelection(t *testing.T) {
	l := newLabel()
	l.Items = []item{{Kind: kindText, X: 4, Y: 8, W: 30, H: 8, labelText: labelText{Value: "标题", FontSize: 5}}}
	e := &editor{label: l, selected: 0, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("加粗"); err != nil {
		t.Fatal(err)
	}
	if e.drag != nil {
		t.Fatal("点击浮动工具栏后开始了拖动")
	}
	if e.selected != 0 {
		t.Fatalf("点击浮动工具栏后取消了选中：%d", e.selected)
	}
}

// TestZoomControl 验证缩放按钮和百分比重置
func TestZoomControl(t *testing.T) {
	l := newLabel()
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("放大"); err != nil {
		t.Fatal(err)
	}
	if e.zoom <= 1 {
		t.Fatalf("放大后缩放比例未变化：%g", e.zoom)
	}
	if err := tt.Click("适合窗口"); err != nil {
		t.Fatal(err)
	}
	if e.zoom != 1 {
		t.Fatalf("适合窗口未恢复缩放：%g", e.zoom)
	}
}

// TestEmptyStateAddsText 验证空白标签上的添加文字按钮
func TestEmptyStateAddsText(t *testing.T) {
	l := newLabel()
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("添加文字"); err != nil {
		t.Fatal(err)
	}
	if len(l.Items) != 1 || e.selected != 0 {
		t.Fatalf("空白状态未添加文字：%d 个元素，选中 %d", len(l.Items), e.selected)
	}
}

// TestPaperSettingsPopover 验证纸张设置卡片会打开尺寸和分辨率设置
func TestPaperSettingsPopover(t *testing.T) {
	l := newLabel()
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if tt.HasText("纸宽") {
		t.Fatal("未点击时已显示纸张设置")
	}
	if err := tt.Click("标签纸张设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("纸宽") || !tt.HasText("打印分辨率") {
		t.Fatal("纸张设置弹层没有显示尺寸或分辨率字段")
	}
}

// TestCanvasClickSelects 验证点击画布中的元素会选中它
func TestCanvasClickSelects(t *testing.T) {
	l := newLabel()
	l.Items = []item{
		{Kind: kindText, X: 20, Y: 12, W: 20, H: 8, labelText: labelText{Value: "标题", FontSize: 4}},
	}
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	canvas, ok := tt.Find("标签画布")
	if !ok {
		t.Fatal("找不到画布")
	}
	paper := e.paperRect(ui.Rect{W: canvas.W, H: canvas.H})
	box := l.Items[0].box(l, paper)
	tt.ClickAt(canvas.X+box.X+box.W/2, canvas.Y+box.Y+box.H/2)
	if e.selected != 0 {
		t.Fatalf("点击元素未选中：%d", e.selected)
	}
}

// TestNewLabelDialog 验证新建对话框可以选规格并创建标签
func TestNewLabelDialog(t *testing.T) {
	l := newLabel()
	e := &editor{label: l, selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	if !e.form.open {
		t.Fatal("点击新建没有打开对话框")
	}
	if err := tt.Click("100 × 50 mm"); err != nil {
		t.Fatal(err)
	}
	if e.form.width != 100 || e.form.height != 50 {
		t.Fatalf("未选中预设规格：%g × %g", e.form.width, e.form.height)
	}
	if err := tt.Click("分辨率 300"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("创建标签"); err != nil {
		t.Fatal(err)
	}
	if e.form.open {
		t.Fatal("创建后对话框没有关闭")
	}
	if e.label.Width != 100 || e.label.Height != 50 || e.label.DPI != 300 {
		t.Fatalf("新建标签规格不对：%g × %g %d DPI", e.label.Width, e.label.Height, e.label.DPI)
	}
}

// TestNewLabelDialogEscape 验证 Esc 关闭新建对话框
func TestNewLabelDialogEscape(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if e.form.open {
		t.Fatal("Esc 没有关闭对话框")
	}
}

// TestCustomSizeEditsPreview 验证自定义尺寸会同步到创建结果
func TestCustomSizeEditsPreview(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("新建"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("自定义尺寸"); err != nil {
		t.Fatal(err)
	}
	if !e.form.custom {
		t.Fatal("点击自定义尺寸没有切换模式")
	}
	e.form.width, e.form.height = 80, 40
	if err := tt.Click("创建标签"); err != nil {
		t.Fatal(err)
	}
	if e.label.Width != 80 || e.label.Height != 40 {
		t.Fatalf("自定义尺寸未生效：%g × %g", e.label.Width, e.label.Height)
	}
}

// TestPrintPreviewOpens 验证打印按钮打开预览对话框并给出检查结果
func TestPrintPreviewOpens(t *testing.T) {
	e := &editor{label: sampleLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("打印"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !e.preview.open {
		t.Fatal("点击打印没有打开预览对话框")
	}
	if !tt.HasText("打印预览") || !tt.HasText("元素都在标签内") {
		t.Fatalf("打印预览内容不完整：%v", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	if e.preview.open {
		t.Fatal("Esc 没有关闭打印预览")
	}
}

// TestPrintPreviewNeedsContent 验证空标签不打开打印预览
func TestPrintPreviewNeedsContent(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	e.openPrintPreview()
	if e.preview.open {
		t.Fatal("空标签不应打开打印预览")
	}
	if e.status != "请先添加标签内容" {
		t.Fatalf("空标签提示不对：%q", e.status)
	}
	_ = tt
}

// TestCornerHandlesResize 验证四角手柄都能调整元素尺寸
func TestCornerHandlesResize(t *testing.T) {
	cases := []struct {
		name         string
		corner       int
		wantX, wantY float64
		wantW, wantH float64
	}{
		{"右下角", 3, 10, 10, 14, 10},
		{"左上角", 0, 8, 8, 14, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newLabel()
			l.Items = []item{{Kind: kindText, X: 10, Y: 10, W: 12, H: 8, labelText: labelText{Value: "标题", FontSize: 4}}}
			e := &editor{label: l, selected: 0, zoom: 1}
			tt := ui.NewTester(e.view, 1280, 800)
			tt.Frame()
			canvas, ok := tt.Find("标签画布")
			if !ok {
				t.Fatal("找不到画布")
			}
			paper := e.paperRect(ui.Rect{W: canvas.W, H: canvas.H})
			box := l.Items[0].box(l, paper)
			handle := handleRects(box)[c.corner]
			step := float32(2) / float32(l.Width) * paper.W
			dx, dy := step, step
			if c.corner == 0 {
				dx, dy = -step, -step
			}
			x, y := handle.X+handle.W/2, handle.Y+handle.H/2
			tt.Press(canvas.X+x, canvas.Y+y)
			tt.Move(canvas.X+x+dx, canvas.Y+y+dy)
			tt.Release(canvas.X+x+dx, canvas.Y+y+dy)
			a := l.Items[0]
			if a.X != c.wantX || a.Y != c.wantY || a.W != c.wantW || a.H != c.wantH {
				t.Fatalf("手柄未调整尺寸：x=%g y=%g w=%g h=%g", a.X, a.Y, a.W, a.H)
			}
		})
	}
}

// TestSettingsDialogOpens 验证「设置」按钮打开通用页面
func TestSettingsDialogOpens(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !e.settings.open || e.settings.page != settingsGeneral {
		t.Fatalf("设置没有打开通用页：open=%v page=%d", e.settings.open, e.settings.page)
	}
	for _, text := range []string{"外观", "偏好", "显示网格", "对齐吸附", "浅色", "深色"} {
		if !tt.HasText(text) {
			t.Fatalf("设置页缺少 %q：%v", text, tt.Texts())
		}
	}
}

// TestSettingsDialogShowsTemplates 验证导航可以切到模板库并列出模板
func TestSettingsDialogShowsTemplates(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("模板库"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if e.settings.page != settingsTemplates {
		t.Fatalf("没有切到模板库：%d", e.settings.page)
	}
	for _, text := range []string{"商品标签", "食品标签", "使用模板"} {
		if !tt.HasText(text) {
			t.Fatalf("模板库缺少 %q：%v", text, tt.Texts())
		}
	}
}

// TestTemplateLibraryApplies 验证选中模板后可以新建同规格的标签
func TestTemplateLibraryApplies(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("模板"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("食品标签"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if e.settings.template != 1 {
		t.Fatalf("没有选中食品标签：%d", e.settings.template)
	}
	if err := tt.Click("使用模板"); err != nil {
		t.Fatal(err)
	}
	if e.settings.open {
		t.Fatal("使用模板后对话框没有关闭")
	}
	if e.label.Width != 60 || e.label.Height != 40 {
		t.Fatalf("模板尺寸不对：%g × %g", e.label.Width, e.label.Height)
	}
	if len(e.label.Items) < 5 {
		t.Fatalf("模板内容没有填入：%d 个元素", len(e.label.Items))
	}
}

// TestTemplateSearchFilters 验证搜索条件会过滤模板卡片
func TestTemplateSearchFilters(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	e.settings = settingsState{open: true, page: settingsTemplates, query: "物流"}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if !tt.HasText("物流标签") {
		t.Fatalf("搜索结果缺少物流标签：%v", tt.Texts())
	}
	if tt.HasText("食品标签") {
		t.Fatalf("搜索结果仍显示食品标签：%v", tt.Texts())
	}
}

// TestNewLabelUsesPreferences 验证新建对话框使用设置中的默认规格
func TestNewLabelUsesPreferences(t *testing.T) {
	restore := prefs
	t.Cleanup(func() { prefs = restore })
	prefs.DefaultWidth, prefs.DefaultHeight, prefs.DefaultDPI = 60, 40, 300
	var f newLabelForm
	f.openNewLabel()
	if f.width != 60 || f.height != 40 || f.dpi != 300 || f.custom {
		t.Fatalf("默认规格没有生效：%g × %g %d custom=%v", f.width, f.height, f.dpi, f.custom)
	}
	prefs.DefaultWidth, prefs.DefaultHeight = 45, 25
	f.openNewLabel()
	if !f.custom {
		t.Fatal("非预设尺寸没有标记为自定义")
	}
}

// TestPreferencesRoundTrip 验证偏好可以保存并重新读取
func TestPreferencesRoundTrip(t *testing.T) {
	restore := prefs
	t.Cleanup(func() {
		prefs = restore
		savePreferences()
	})
	prefs = preferences{DefaultWidth: 70, DefaultHeight: 45, DefaultDPI: 600}
	if err := os.WriteFile(settingsFile(), []byte("{}"), 0o644); err != nil {
		t.Logf("预置设置文件失败：%v", err)
	}
	savePreferences()
	prefs = defaultPreferences()
	loadPreferences()
	if prefs.DefaultWidth != 70 || prefs.DefaultHeight != 45 || prefs.DefaultDPI != 600 {
		t.Fatalf("偏好没有恢复：%+v", prefs)
	}
	if prefs.ShowGrid || prefs.SnapGuides {
		t.Fatalf("开关状态没有恢复：%+v", prefs)
	}
}

// TestTemplateWithBarcodeApplies 验证含条码的模板在无图形上下文时也能创建
func TestTemplateWithBarcodeApplies(t *testing.T) {
	e := &editor{label: newLabel(), selected: -1, zoom: 1}
	e.settings = settingsState{open: true, page: settingsTemplates}
	e.useTemplate(labelTemplates[0])
	if e.label.Width != 50 || e.label.Height != 30 {
		t.Fatalf("模板尺寸不对：%g × %g", e.label.Width, e.label.Height)
	}
	tt := ui.NewTester(e.view, 1280, 800)
	tt.Frame()
	if e.settings.open {
		t.Fatal("使用模板后对话框没有关闭")
	}
}
