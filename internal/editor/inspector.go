package editor

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

// inspector 显示所选元素的属性，未选中时显示标签尺寸和打印设置
func (e *editor) inspector(c *ui.Context) {
	ui.Column(c).Width(inspectorWidth).Shrink(0).FillHeight().Background(colors.Panel).
		BorderWidth(0, 0, 0, 1).BorderColor(colors.Divider).Children(func() {
		a := e.selectedItem()
		if a == nil {
			e.inspectorHeader(c.Key("head"), i18n.Tr("inspector.label"), iconPaper, false)
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
					sectionHeading(c, i18n.Tr("inspector.content"))
					if ui.TextInput(c.Key("value"), &a.Value).Placeholder(i18n.Tr("inspector.placeholder")).
						Label(i18n.Tr("inspector.encoded")).FillWidth().Height(34).Radius(7).FontSize(12).
						Border(1, colors.FieldBorder).Background(colors.Panel).Changed() {
						a.prepareSymbol(e.graphics)
						e.changed()
					}
					if a.issue != nil {
						ui.Text(c.Key("issue"), a.issue.Error()).FontSize(11).TextColor(c.Theme().Danger).Wrap()
					}
				case kindTable:
					sectionHeading(c, i18n.Tr("inspector.table"))
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
		BorderWidth(0, 0, 1, 0).BorderColor(colors.Hairline).Children(func() {
		ui.Row(c.Key("icon")).Size(34, 34).Radius(8).Shrink(0).Background(colors.Selected).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon).Size(18, 18).TextColor(colors.Accent)
		})
		ui.Text(c, title).FontSize(13.5).FontWeight(650).TextColor(colors.Heading).Grow(1)
		if !removable {
			return
		}
		button := ui.ButtonBase(c.Key("delete")).Size(30, 30).Radius(7).Shrink(0).
			Role(ui.RoleButton).Label(i18n.Tr("inspector.delete")).Tooltip(i18n.Tr("inspector.delete"))
		ink := colors.Faint
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

// labelProperties 显示标签尺寸和打印分辨率
func (e *editor) labelProperties(c *ui.Context) {
	l := e.label
	width, height, dpi := l.Width, l.Height, l.DPI
	sectionHeading(c, i18n.Tr("inspector.size"))
	ui.Row(c.Key("size")).Gap(8).Children(func() {
		inspectorDimension(c, i18n.Tr("inspector.width"), &l.Width, minimumSide, maximumSide)
		inspectorDimension(c, i18n.Tr("inspector.height"), &l.Height, minimumSide, maximumSide)
	})

	hairline(c.Key("divider"))

	sectionHeading(c, i18n.Tr("inspector.print"))
	resolutions := [...]int{203, 300, 600}
	selected := max(0, slices.Index(resolutions[:], l.DPI))
	ui.Column(c.Key("dpi")).Gap(5).Children(func() {
		ui.Text(c, i18n.Tr("inspector.resolution")).FontSize(10).TextColor(colors.Muted)
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
}

// positionProperties 显示元素的位置和毫米尺寸
func (e *editor) positionProperties(c *ui.Context, a *item) {
	l := e.label
	x, y, width, height := a.X, a.Y, a.W, a.H
	sectionHeading(c, i18n.Tr("inspector.position"))
	ui.Row(c.Key("position")).Gap(8).Children(func() {
		inspectorDimension(c, "X", &a.X, 0, l.Width-a.W)
		inspectorDimension(c, "Y", &a.Y, 0, l.Height-a.H)
	})
	ui.Row(c.Key("size")).Gap(8).Children(func() {
		widthName, heightName := i18n.Tr("inspector.width"), i18n.Tr("inspector.height")
		if a.Kind == kindLine {
			widthName, heightName = i18n.Tr("inspector.length"), i18n.Tr("table.stroke")
		}
		inspectorDimension(c, widthName, &a.W, minimumItem, l.Width-a.X)
		inspectorDimension(c, heightName, &a.H, a.minimumHeight(), l.Height-a.Y)
	})
	if x != a.X || y != a.Y || width != a.W || height != a.H {
		a.constrain(l)
		e.changed()
	}
}
