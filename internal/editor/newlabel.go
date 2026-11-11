package editor

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

const (
	// newLabelDialogWidth 是新建标签对话框的内容宽度
	newLabelDialogWidth = 880
	// newLabelSettingsWidth 是对话框右侧预览与设置的宽度
	newLabelSettingsWidth = 300
	// newLabelPresetHeight 是规格卡片的固定高度
	newLabelPresetHeight = 80
	// newLabelPreviewHeight 是纸张预览区的高度
	newLabelPreviewHeight = 168
)

// newLabelPreset 是对话框中的一个常用标签规格
type newLabelPreset struct {
	label         string
	width, height float64
}

// newLabelPresets 是按设计稿顺序排列的常用规格
var newLabelPresets = [...]newLabelPreset{
	{"50 × 30 mm", 50, 30},
	{"40 × 30 mm", 40, 30},
	{"60 × 40 mm", 60, 40},
	{"100 × 50 mm", 100, 50},
	{"40 × 60 mm", 40, 60},
}

// newLabelForm 保存新建标签对话框的输入状态
type newLabelForm struct {
	open          bool
	width, height float64
	dpi           int
	name          string
	custom        bool
}

// openNewLabel 按设置中的默认规格打开新建对话框
func (f *newLabelForm) openNewLabel() {
	*f = newLabelForm{
		open: true, width: prefs.DefaultWidth, height: prefs.DefaultHeight,
		dpi: prefs.DefaultDPI, name: i18n.Tr("header.untitled"),
	}
	f.custom = f.presetIndex() < 0
}

// presetIndex 返回与当前尺寸匹配的规格下标，自定义尺寸返回 -1
func (f *newLabelForm) presetIndex() int {
	if f.custom {
		return -1
	}
	for i, preset := range newLabelPresets {
		if preset.width == f.width && preset.height == f.height {
			return i
		}
	}
	return -1
}

// choosePreset 选中一个常用规格
func (f *newLabelForm) choosePreset(i int) {
	f.width, f.height = newLabelPresets[i].width, newLabelPresets[i].height
	f.custom = false
}

// createLabel 按对话框中的规格新建标签
func (e *editor) createLabel() {
	f := &e.form
	f.open = false
	l := &label{Width: roundMM(f.width), Height: roundMM(f.height), DPI: f.dpi, title: strings.TrimSpace(f.name)}
	if e.tabs == nil {
		e.label, e.selected, e.drag = l, -1, nil
		e.textEdit, e.cells, e.inline = nil, nil, false
		return
	}
	e.tabs.newTab(l)
}

// newLabelDialog 在选择新建时显示规格对话框
func (e *editor) newLabelDialog(c *ui.Context) {
	f := &e.form
	ui.Modal(c, &f.open, func() {
		ui.Column(c.Key("dialog")).Width(newLabelDialogWidth).Children(func() {
			e.dialogHeading(c.Key("heading"))
			hairline(c.Key("rule"))
			ui.Row(c.Key("body")).Gap(26).Padding(16, 0, 18, 0).AlignItems(ui.Start).Children(func() {
				e.dialogSpecs(c.Key("specs"))
				e.dialogSettings(c.Key("settings"))
			})
			hairline(c.Key("rule-2"))
			ui.Row(c.Key("actions")).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Box(c.Key("spacer")).Grow(1)
				plainAction(c.Key("cancel"), i18n.Tr("new.cancel"), func() { f.open = false })
				accentAction(c.Key("create"), i18n.Tr("new.create"), iconChevron, true, func() { e.createLabel() })
			})
		})
	})
}

// dialogHeading 显示对话框标题和关闭按钮
func (e *editor) dialogHeading(c *ui.Context) {
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, i18n.Tr("new.title")).FontSize(19).FontWeight(650).TextColor(colors.Heading).Grow(1)
		iconAction(c.Key("close"), i18n.Tr("new.close"), iconClose, func() { e.form.open = false })
	})
}

// dialogSpecs 显示常用规格与自定义尺寸卡片
func (e *editor) dialogSpecs(c *ui.Context) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
		sectionHeading(c, i18n.Tr("new.specs"))
		ui.Column(c.Key("grid")).Gap(10).Children(func() {
			for i := 0; i < len(newLabelPresets); i += 2 {
				ui.Row(c.Key(i)).Gap(10).Children(func() {
					for _, preset := range []int{i, i + 1} {
						if preset < len(newLabelPresets) {
							e.presetCard(c.Key(preset), preset)
						} else {
							e.customCard(c.Key("custom"))
						}
					}
				})
			}
		})
	})
}

// presetCard 显示一个常用规格卡片，选中的卡片为主色描边
func (e *editor) presetCard(c *ui.Context, index int) {
	preset := newLabelPresets[index]
	f := &e.form
	selected := f.presetIndex() == index
	ink, fill, border := colors.Text, colors.Card, colors.CardBorder
	if selected {
		ink, fill, border = colors.Accent, colors.AccentSoft, colors.Accent
	}
	width := float32(1)
	if selected {
		width = 1.5
	}
	card := ui.ButtonBase(c).Grow(1).Height(newLabelPresetHeight).Padding(12).Gap(12).
		Radius(10).AlignItems(ui.Center).Role(ui.RoleButton).Label(preset.label).
		AccessSelected(selected).Cursor(ui.CursorPointer).
		BorderWidth(width).BorderColor(border).Background(fill)
	if card.Hovered() && !selected {
		card.BorderColor(colors.BorderHover)
	}
	card.Children(func() {
		ui.Row(c.Key("chip")).Size(46, 42).Radius(7).Shrink(0).Background(colors.Chip).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			scale := min(float32(34)/float32(preset.width), float32(30)/float32(preset.height))
			ui.Box(c.Key("sheet")).Width(max(8, float32(preset.width)*scale)).
				Height(max(6, float32(preset.height)*scale)).Background(colors.Paper).
				BorderWidth(1).BorderColor(colors.BorderHover).
				Shadow(0, 1, 2, 0, colors.ShadowSoft)
		})
		ui.Text(c, preset.label).FontSize(13).FontWeight(600).SingleLine().Grow(1).TextColor(ink)
		if selected {
			ui.Row(c.Key("badge")).Absolute().Right(9).Top(9).Size(17, 17).Radius(9).
				Background(colors.Primary).Justify(ui.Center).AlignItems(ui.Center).
				Children(func() {
					ui.Icon(c, iconCheck).Size(10, 10).TextColor(colors.OnAccent)
				})
		}
	})
	if card.Clicked() {
		f.choosePreset(index)
	}
}

// customCard 显示虚线边框的自定义尺寸卡片
func (e *editor) customCard(c *ui.Context) {
	f := &e.form
	selected := f.custom
	fill, border := colors.FieldBg, colors.BorderHover
	if selected {
		fill, border = colors.AccentSoft, colors.Accent
	}
	card := ui.ButtonBase(c).Grow(1).Height(newLabelPresetHeight).Padding(12).Gap(12).
		Radius(10).AlignItems(ui.Center).Role(ui.RoleButton).Label(i18n.Tr("new.custom")).
		AccessSelected(selected).Cursor(ui.CursorPointer).
		BorderWidth(1).BorderColor(border).Background(fill)
	if !selected {
		card.BorderStyle(ui.BorderDashed)
	}
	if card.Hovered() && !selected {
		card.BorderColor(colors.BorderStrong).Background(colors.Panel)
	}
	card.Children(func() {
		ui.Row(c.Key("chip")).Size(46, 42).Radius(7).Shrink(0).Background(colors.Chip).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconPlus).Size(18, 18).TextColor(colors.Muted)
		})
		ui.Text(c, i18n.Tr("new.custom")).FontSize(13).FontWeight(600).SingleLine().Grow(1).
			TextColor(colors.Body)
	})
	if card.Clicked() {
		f.custom = true
	}
}

// dialogSettings 显示纸张预览、毫米尺寸、分辨率和名称
func (e *editor) dialogSettings(c *ui.Context) {
	f := &e.form
	ui.Column(c).Width(newLabelSettingsWidth).Shrink(0).Gap(12).Children(func() {
		sectionHeading(c, i18n.Tr("new.preview"))
		ui.Box(c.Key("preview")).Height(newLabelPreviewHeight).Radius(12).
			BorderWidth(1).BorderColor(colors.Divider).Background(colors.Stage).
			Clip().Draw(e.paintPaperPreview)

		ui.Row(c.Key("size")).Gap(8).Children(func() {
			inspectorDimension(c, i18n.Tr("new.width"), &f.width, minimumSide, maximumSide)
			inspectorDimension(c, i18n.Tr("new.height"), &f.height, minimumSide, maximumSide)
		})

		resolutions := [...]int{203, 300, 600}
		selected := max(0, slices.Index(resolutions[:], f.dpi))
		ui.Column(c.Key("dpi")).Gap(5).Children(func() {
			ui.Text(c, i18n.Tr("new.resolution")).FontSize(10).TextColor(colors.Muted)
			segmentedRow(c.Key("segments"), i18n.Tr("new.resolution"), []string{"203", "300", "600"}, selected, func(i int) {
				f.dpi = resolutions[i]
			})
		})

		ui.Column(c.Key("name")).Gap(5).Children(func() {
			ui.Text(c, i18n.Tr("new.name")).FontSize(10).TextColor(colors.Muted)
			ui.Row(c.Key("field")).Height(34).Radius(7).BorderWidth(1).
				BorderColor(colors.FieldBorder).Background(colors.Panel).Children(func() {
				ui.TextInputBase(c.Key("entry"), &f.name).Grow(1).MinWidth(0).Height(32).
					Padding(0, 10).FontSize(12).Label(i18n.Tr("new.namePlaceholder"))
			})
		})
	})
}

// paintPaperPreview 绘制点阵底和按比例缩放的纸张示意
func (e *editor) paintPaperPreview(p *ui.Painter, r ui.Rect) {
	p.Fill(r, colors.Stage, 0)
	for y := r.Y + 8; y < r.Y+r.H; y += 16 {
		for x := r.X + 8; x < r.X+r.W; x += 16 {
			p.Fill(ui.Rect{X: x - 1, Y: y - 1, W: 2, H: 2}, colors.SketchDot, 1)
		}
	}
	f := &e.form
	width, height := max(f.width, 1), max(f.height, 1)
	scale := min((r.W-64)/float32(width), (r.H-52)/float32(height))
	paper := ui.Rect{
		X: r.X + (r.W-float32(width)*scale)/2,
		Y: r.Y + (r.H-float32(height)*scale)/2,
		W: float32(width) * scale,
		H: float32(height) * scale,
	}
	p.Shadow(paper, 0, 0, 10, 24, -8, colors.ShadowStrong)
	p.Fill(paper, colors.Paper, 0)
	p.Stroke(paper, ui.Hex("#D5DEE8"), 0, 1)
	paintDashedRect(p, ui.Rect{X: paper.X + 10, Y: paper.Y + 10, W: paper.W - 20, H: paper.H - 20},
		ui.Hex("#DDE5EF"))
}

// paintDashedRect 用短线段描出虚线矩形
func paintDashedRect(p *ui.Painter, r ui.Rect, c ui.Color) {
	const dash, gap = 4, 4
	for x := r.X; x < r.X+r.W; x += dash + gap {
		end := min(x+dash, r.X+r.W)
		p.Line(x, r.Y, end, r.Y, 1, c)
		p.Line(x, r.Y+r.H, end, r.Y+r.H, 1, c)
	}
	for y := r.Y; y < r.Y+r.H; y += dash + gap {
		end := min(y+dash, r.Y+r.H)
		p.Line(r.X, y, r.X, end, 1, c)
		p.Line(r.X+r.W, y, r.X+r.W, end, 1, c)
	}
}

// plainAction 显示对话框底部的无边框文字按钮
func plainAction(c *ui.Context, label string, action func()) {
	button := ui.ButtonBase(c).Height(36).Padding(0, 16).Radius(8).Shrink(0).
		AlignItems(ui.Center).Role(ui.RoleButton).Label(label).Cursor(ui.CursorPointer)
	if button.Hovered() {
		button.Background(colors.Hover)
	}
	button.Children(func() {
		ui.Text(c, label).FontSize(12.5).FontWeight(500).TextColor(colors.Body2)
	})
	if button.Clicked() {
		action()
	}
}
