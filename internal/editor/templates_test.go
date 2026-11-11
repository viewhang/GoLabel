package editor

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestTemplateDialogScrollsWithinBounds 验证较多模板在弹框内滚动且预览操作始终可见
func TestTemplateDialogScrollsWithinBounds(t *testing.T) {
	original := labelTemplates
	t.Cleanup(func() { labelTemplates = original })
	for _, count := range []int{7, 17} {
		for _, size := range [][2]int{{1280, 800}, {980, 660}} {
			t.Run(fmt.Sprintf("%d个模板/%d×%d", count, size[0], size[1]), func(t *testing.T) {
				labelTemplates = slices.Clone(original)
				for len(labelTemplates) < count {
					l := newLabel()
					labelTemplates = append(labelTemplates, labelTemplate{
						name: fmt.Sprintf("自定义模板 %02d", len(labelTemplates)), category: "custom",
						width: l.Width, height: l.Height, dpi: l.DPI, custom: l,
					})
				}
				e := &editor{label: newLabel(), selected: -1, zoom: 1}
				e.settings = settingsState{open: true, page: settingsTemplates}
				tt := ui.NewTester(e.view, size[0], size[1])
				tt.Frame()
				dialog, ok := tt.AccessByRole(ui.RoleDialog, "")
				if !ok {
					t.Fatal("找不到模板弹框")
				}
				close, ok := tt.Find("关闭")
				if !ok {
					t.Fatal("找不到弹框关闭按钮")
				}
				inset := close.Y - dialog.Bounds.Y
				button, ok := tt.Find("使用模板")
				if !ok || button.H == 0 || button.Y+button.H+inset > dialog.Bounds.Y+dialog.Bounds.H {
					saveImage(t, "golabel-template-overflow", tt)
					t.Fatalf("使用模板按钮超出弹框：按钮 %+v，弹框 %+v", button, dialog.Bounds)
				}
				first, ok := tt.AccessByRole(ui.RoleButton, labelTemplates[0].title())
				if !ok {
					t.Fatal("找不到首个模板卡片")
				}
				tt.Scroll(first.Bounds.X+first.Bounds.W/2, first.Bounds.Y+20, 0, 3000)
				lastIndex := len(labelTemplates) - 1
				lastName := labelTemplates[lastIndex].title()
				last, ok := tt.AccessByRole(ui.RoleButton, lastName)
				if !ok || last.Bounds.H == 0 || last.Bounds.Y+last.Bounds.H > dialog.Bounds.Y+dialog.Bounds.H {
					t.Fatalf("滚动后最后一个模板没有在弹框内显示：%+v", last.Bounds)
				}
				after, _ := tt.Find("使用模板")
				if after != button {
					t.Fatal("滚动模板列表时移动了预览操作按钮")
				}
				if err := tt.Click(lastName); err != nil {
					t.Fatal(err)
				}
				if e.settings.template != lastIndex {
					t.Fatal("无法选中滚动后的模板")
				}
				saveImage(t, fmt.Sprintf("golabel-template-fixed-%d-%d", count, size[0]), tt)
				if err := tt.Click("使用模板"); err != nil {
					t.Fatal(err)
				}
				if e.settings.open || e.label.title != lastName {
					t.Fatal("没有正确使用所选模板并关闭弹框")
				}
			})
		}
	}
}
