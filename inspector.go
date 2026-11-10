package main

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// inspector 显示所选元素的属性，未选中时显示标签尺寸和打印设置
func (e *editor) inspector(c *ui.Context) {
	ui.Column(c).Width(inspectorWidth).Shrink(0).FillHeight().Background(ui.Hex(colorPanel)).
		BorderWidth(0, 0, 0, 1).BorderColor(ui.Hex(colorDivider)).Children(func() {
		a := e.selectedItem()
		if a == nil {
			e.inspectorHeader(c.Key("head"), "标签", iconPaper, false)
			ui.Scroll(c.Key("body")).Grow(1).MinHeight(0).Children(func() {
				ui.Column(c.Key("fields")).Padding(panelPadding).Gap(8).Children(func() {
					e.labelProperties(c.Key("label"))
				})
			})
			return
		}
		if e.inspectorHeader(c.Key("head"), kindTitle(a.Kind), layerIcon(a.Kind), true) {
			e.remove()
			return
		}
		ui.Scroll(c.Key("body")).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key("fields")).Padding(panelPadding).Gap(8).Children(func() {
				switch a.Kind {
				case kindText:
					if e.textProperties(c.Key("text"), &a.labelText) {
						e.changed()
					}
				case kindQR, kindBarcode:
					sectionHeading(c, "内容")
					if ui.TextInput(c.Key("value"), &a.Value).Placeholder("输入内容").
						Label("编码内容").FillWidth().Height(34).Radius(7).FontSize(12).
						Border(1, ui.Hex(colorFieldBorder)).Background(ui.Hex(colorPanel)).Changed() {
						a.prepareSymbol(e.graphics)
						e.changed()
					}
					if a.issue != nil {
						ui.Text(c.Key("issue"), a.issue.Error()).FontSize(11).TextColor(c.Theme().Danger).Wrap()
					}
				case kindTable:
					sectionHeading(c, "表格")
					e.tableProperties(c.Key("table"), a)
				}
				hairline(c.Key("divider"))
				e.positionProperties(c.Key("geometry"), a)
			})
		})
	})
}

// inspectorHeader 显示属性栏顶部图标、名称和删除按钮
func (e *editor) inspectorHeader(c *ui.Context, title string, icon *ui.SVG, removable bool) bool {
	t := c.Theme()
	remove := false
	ui.Row(c).Height(64).Shrink(0).Padding(0, panelPadding).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex(colorHairline)).Children(func() {
		ui.Row(c.Key("icon")).Size(34, 34).Radius(8).Shrink(0).Background(ui.Hex(colorChip)).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(18, 18).TextColor(ui.Hex(colorAccent))
		})
		ui.Text(c, title).FontSize(13.5).FontWeight(650).TextColor(ui.Hex(colorHeading)).Grow(1)
		if !removable {
			return
		}
		button := ui.ButtonBase(c.Key("delete")).Size(30, 30).Radius(7).Shrink(0).
			Role(ui.RoleButton).Label("删除元素").Tooltip("删除元素")
		ink := ui.Hex(colorFaint)
		if button.Hovered() {
			button.Background(t.Danger.Alpha(0.10))
			ink = t.Danger
		}
		button.Children(func() {
			ui.Icon(c, iconTrash).Size(15, 15).TextColor(ink)
		})
		remove = button.Clicked()
	})
	return remove
}

// labelProperties 显示标签尺寸、打印分辨率和元素数量
func (e *editor) labelProperties(c *ui.Context) {
	l := e.label
	width, height, dpi := l.Width, l.Height, l.DPI
	sectionHeading(c, "尺寸")
	ui.Row(c.Key("size")).Gap(8).Children(func() {
		inspectorDimension(c, "宽度", &l.Width, minimumSide, maximumSide)
		inspectorDimension(c, "高度", &l.Height, minimumSide, maximumSide)
	})

	hairline(c.Key("divider"))

	sectionHeading(c, "打印")
	resolutions := [...]int{203, 300, 600}
	selected := max(0, slices.Index(resolutions[:], l.DPI))
	ui.Column(c.Key("dpi")).Gap(5).Children(func() {
		ui.Text(c, "分辨率").FontSize(10).TextColor(ui.Hex(colorMuted))
		optionRow(c.Key("segments"), []string{"203", "300", "600"}, selected, func(i int) {
			l.DPI = resolutions[i]
		})
	})

	if width != l.Width || height != l.Height {
		for i := range l.Items {
			l.Items[i].constrain(l)
		}
	}
	if width != l.Width || height != l.Height || dpi != l.DPI {
		e.changed()
	}

	hairline(c.Key("divider-2"))

	ui.Row(c.Key("summary")).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "画布中的元素").FontSize(12).TextColor(ui.Hex(colorMuted)).Grow(1)
		ui.Textf(c, "%d", len(l.Items)).FontSize(12).FontWeight(600).TextColor(ui.Hex(colorText))
	})
}

// positionProperties 显示元素的位置和毫米尺寸
func (e *editor) positionProperties(c *ui.Context, a *item) {
	l := e.label
	x, y, width, height := a.X, a.Y, a.W, a.H
	sectionHeading(c, "位置与尺寸")
	ui.Row(c.Key("position")).Gap(8).Children(func() {
		inspectorDimension(c, "X", &a.X, 0, l.Width-a.W)
		inspectorDimension(c, "Y", &a.Y, 0, l.Height-a.H)
	})
	ui.Row(c.Key("size")).Gap(8).Children(func() {
		widthName, heightName := "宽度", "高度"
		if a.Kind == kindLine {
			widthName, heightName = "长度", "线宽"
		}
		inspectorDimension(c, widthName, &a.W, minimumItem, l.Width-a.X)
		inspectorDimension(c, heightName, &a.H, a.minimumHeight(), l.Height-a.Y)
	})
	if x != a.X || y != a.Y || width != a.W || height != a.H {
		a.constrain(l)
		e.changed()
	}
}
