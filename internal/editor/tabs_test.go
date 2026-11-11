package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
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
