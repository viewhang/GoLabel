package editor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// TestDocumentTabsBar 验证操作提示不影响保存状态，标签栏加号能打开新建对话框
func TestDocumentTabsBar(t *testing.T) {
	tabs := &documentTabs{}
	first := &editor{label: newLabel(), tabs: tabs, selected: -1, zoom: 1}
	first.label.title = "桂花乌龙标签"
	second := &editor{label: newLabel(), tabs: tabs, selected: -1, zoom: 1}
	second.label.title = "咖啡豆标签"
	tabs.editors = []*editor{first, second}
	tabs.active = first
	first.status = "已保存到我的模板"

	tt := ui.NewTester(first.view, 980, 660)
	for _, tab := range tabs.editors {
		name := tab.tabName()
		node, ok := tt.AccessByRole(ui.RoleTab, name)
		if !ok || node.Selected != (tab == first) {
			t.Fatalf("标签页 %q 的选中状态不对", name)
		}
		if _, ok := tt.Find("关闭 " + name); !ok {
			t.Fatalf("标签页 %q 缺少关闭按钮", name)
		}
	}
	if !tt.HasText("草稿") {
		t.Fatal("未保存过的文档没有显示草稿状态")
	}
	if !tt.HasText("已保存到我的模板") {
		t.Fatal("没有显示保存模板的操作提示")
	}
	if err := tt.Click("新建标签页"); err != nil {
		t.Fatal(err)
	}
	if !first.form.open {
		t.Fatal("加号没有打开新建标签对话框")
	}
}

// TestTabSwitchPreservesEditingState 验证单窗口切换保留富文本、撤销、选区及画布状态
func TestTabSwitchPreservesEditingState(t *testing.T) {
	tabs := &documentTabs{}
	first := tabs.newTab(sampleLabel())
	first.label.title = "文字标签"
	first.selectItem(0)
	first.zoom, first.pan = 1.5, ui.Point{X: 40, Y: -20}
	doc := first.textEdit.doc
	doc.Edit(func(edit *richtext.Edit) { edit.ReplaceString(0, 2, "茉莉", richtext.Attributes{}) })
	doc.SetSelection(1, 3)
	second := tabs.newTab(sampleLabel())
	second.label.title = "表格标签"
	second.selected = 4
	second.cells = &tableSelection{start: cellRange{}, end: cellRange{Bottom: 1, Right: 1}}
	second.zoom, second.pan = 0.75, ui.Point{X: -10, Y: 15}
	tabs.selectTab(first)
	tt := ui.NewTester(tabs.view, 1280, 800)
	canvas := first.canvas
	if err := tt.Click("表格标签"); err != nil {
		t.Fatal(err)
	}
	if tabs.active != second || second.selected != 4 || second.cells.end.Right != 1 || second.zoom != 0.75 {
		t.Fatal("切换后表格标签的选择或缩放状态丢失")
	}
	if err := tt.Click("文字标签"); err != nil {
		t.Fatal(err)
	}
	if tabs.active != first || first.canvas != canvas || first.textEdit.doc != doc || first.selected != 0 ||
		first.zoom != 1.5 || first.pan != (ui.Point{X: 40, Y: -20}) {
		t.Fatal("切换回来后编辑控件或画布状态被重建")
	}
	if anchor, caret := doc.Selection(); anchor != 1 || caret != 3 {
		t.Fatalf("富文本选区丢失：%d、%d", anchor, caret)
	}
	if first.label.Items[0].textContent().PlainText() != "茉莉乌龙" || !first.documentEdited() {
		t.Fatal("离开标签时没有写回文字或未保留修改标记")
	}
	if !doc.CanUndo() {
		t.Fatal("切换标签后撤销历史丢失")
	}
	doc.Undo()
	first.flushTextEdit()
	if first.label.Items[0].textContent().PlainText() != "桂花乌龙" {
		t.Fatal("切换标签后撤销未还原原文字")
	}
}

// TestTabCloseCancellation 验证取消关闭和取消另存为均保留未保存标签
func TestTabCloseCancellation(t *testing.T) {
	ask, choose := askDocumentChanges, chooseLabelPath
	t.Cleanup(func() { askDocumentChanges, chooseLabelPath = ask, choose })
	tabs := &documentTabs{}
	first := tabs.newTab(sampleLabel())
	first.changed()
	second := tabs.newTab(newLabel())
	askDocumentChanges = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		return mygo.MessageResult{Button: 2}, nil
	}
	tabs.closeTab(first)
	if len(tabs.editors) != 2 || tabs.active != first || !first.documentEdited() {
		t.Fatal("取消关闭后丢失了标签或修改")
	}
	askDocumentChanges = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		return mygo.MessageResult{Button: 0}, nil
	}
	chooseLabelPath = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	tabs.closeTab(first)
	if len(tabs.editors) != 2 || !first.documentEdited() {
		t.Fatal("取消保存路径后仍关闭了标签")
	}
	askDocumentChanges = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		return mygo.MessageResult{Button: 1}, nil
	}
	tabs.closeTab(first)
	if len(tabs.editors) != 1 || tabs.active != second {
		t.Fatal("确认放弃后未选中相邻标签")
	}
	tabs.closeTab(second)
	if len(tabs.editors) != 1 || tabs.active == second || tabs.active.label == nil {
		t.Fatal("关闭最后一个标签后没有保留主窗口的空白标签")
	}
}

// TestTabFileOperations 验证保存目标、同文件复用及保存失败时的未保存状态
func TestTabFileOperations(t *testing.T) {
	tabs := &documentTabs{}
	first := tabs.newTab(sampleLabel())
	path := filepath.Join(t.TempDir(), "商品.golabel")
	first.changed()
	if err := first.saveTo(path); err != nil {
		t.Fatal(err)
	}
	if first.documentEdited() || first.document.path != path || first.tabName() != "商品" {
		t.Fatal("保存后路径、标签名或保存标记不正确")
	}
	second := tabs.newTab(sampleLabel())
	if err := tabs.open(path); err != nil {
		t.Fatal(err)
	}
	if len(tabs.editors) != 2 || tabs.active != first {
		t.Fatal("打开同一文件创建了重复标签")
	}
	if err := second.saveTo(path); err == nil {
		t.Fatal("允许另一个标签覆盖已经打开的文件")
	}
	second.changed()
	if err := second.saveTo(filepath.Join(path, "无效.golabel")); err == nil || !second.documentEdited() {
		t.Fatal("保存失败后丢失了未保存状态")
	}
	loaded := &documentTabs{}
	if err := loaded.open(path); err != nil {
		t.Fatal(err)
	}
	if got := loaded.active.label.Items[0].Value; got != "桂花乌龙" {
		t.Fatalf("重新打开后内容有误：%q", got)
	}
}

// TestWindowCloseKeepsTabsOnCancel 验证关闭主窗口途中取消不会丢弃已检查的标签
func TestWindowCloseKeepsTabsOnCancel(t *testing.T) {
	ask := askDocumentChanges
	t.Cleanup(func() { askDocumentChanges = ask })
	tabs := &documentTabs{}
	first, second := tabs.newTab(sampleLabel()), tabs.newTab(sampleLabel())
	first.changed()
	second.changed()
	choices := []int{1, 2}
	askDocumentChanges = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		choice := choices[0]
		choices = choices[1:]
		return mygo.MessageResult{Button: choice}, nil
	}
	var event mygo.CloseEvent
	tabs.closeWindow(&event)
	if !event.DefaultPrevented() || len(tabs.editors) != 2 || !first.documentEdited() || !second.documentEdited() {
		t.Fatal("取消退出时关闭了标签或丢失了未保存内容")
	}
}

// TestExternalFileChanges 验证外部修改、重命名和删除不会覆盖本地未保存内容
func TestExternalFileChanges(t *testing.T) {
	tabs := &documentTabs{}
	e := tabs.newTab(newLabel())
	path := filepath.Join(t.TempDir(), "原文件.golabel")
	if err := e.saveTo(path); err != nil {
		t.Fatal(err)
	}
	replacement := newLabel()
	replacement.Width = 60
	other := (&documentTabs{}).newTab(replacement)
	if err := other.saveTo(path); err != nil {
		t.Fatal(err)
	}
	e.checkFile()
	if e.label.Width != 60 || e.documentEdited() {
		t.Fatal("未编辑标签没有读取外部修改")
	}
	e.label.Width = 70
	e.changed()
	replacement.Width = 80
	if err := other.save(); err != nil {
		t.Fatal(err)
	}
	e.checkFile()
	if e.label.Width != 70 || !e.document.conflict {
		t.Fatal("外部修改覆盖了当前未保存内容")
	}
	ask := askDocumentChanges
	t.Cleanup(func() { askDocumentChanges = ask })
	askDocumentChanges = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		return mygo.MessageResult{Button: 1}, nil
	}
	if err := e.save(); !errors.Is(err, mygo.ErrCanceled) || !e.documentEdited() {
		t.Fatal("取消覆盖外部文件后丢失了修改")
	}
	e.document.edited = false
	e.checkFile()
	renamed := filepath.Join(filepath.Dir(path), "新文件.golabel")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	e.checkFile()
	if e.document.path != renamed || e.document.deleted {
		t.Fatal("外部重命名后未跟随同一文件")
	}
	if err := os.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	e.checkFile()
	if !e.document.deleted {
		t.Fatal("文件删除后没有保留删除状态")
	}
}

// TestDocumentRecoveryRoundTrip 验证未保存标签的恢复内容、名称和保存后的副本清理
func TestDocumentRecoveryRoundTrip(t *testing.T) {
	originalSettings := settingsFile
	dir := t.TempDir()
	settingsFile = func() string { return filepath.Join(dir, "settings.json") }
	t.Cleanup(func() { settingsFile = originalSettings })
	tabs := &documentTabs{}
	e := tabs.newTab(sampleLabel())
	e.label.title = "待恢复标签"
	e.changed()
	if err := e.saveRecovery(); err != nil {
		t.Fatal(err)
	}
	path := e.document.recoveryPath
	restored := &documentTabs{}
	restored.initial = restored.newTab(newLabel())
	restored.restoreRecovery()
	if len(restored.editors) != 1 || restored.active.documentName() != "待恢复标签" ||
		!restored.active.documentEdited() || restored.active.label.Items[0].Value != "桂花乌龙" {
		t.Fatal("未保存标签没有完整恢复")
	}
	if err := restored.active.saveTo(filepath.Join(dir, "已保存.golabel")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("保存后仍留有恢复副本")
	}
}
