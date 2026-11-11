package editor

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

// numberDraft 保存尺寸输入中的文字和已提交数值
type numberDraft struct {
	text  string
	value float64
}

// inspectorDimension 显示属性栏中的一个毫米尺寸字段
func inspectorDimension(c *ui.Context, name string, value *float64, low, high float64) {
	field := c.Key(name)
	decimals := 1
	if v := *value; v == math.Trunc(v) {
		decimals = 0
	}
	ui.Column(field).Grow(1).MinWidth(0).Gap(5).Children(func() {
		ui.Text(field, name).FontSize(10).TextColor(colors.Muted)
		inspectorNumber(field, name, "mm", 0, value, low, high, decimals, false)
	})
}

// inspectorNumber 显示指定精度和单位的紧凑步进输入框
func inspectorNumber(c *ui.Context, name, unit string, width float32, value *float64, low, high float64, decimals int, stepper bool) {
	high = max(low, high)
	scale := math.Pow10(decimals)
	step := 1 / scale
	format := func(value float64) string { return strconv.FormatFloat(value, 'f', decimals, 64) }
	label := name
	if unit != "" {
		label += i18n.Trf("unit.suffix", unit)
	}
	row := ui.Row(c.Key("row")).Height(34).MinWidth(0).AlignItems(ui.Center).Radius(7).Clip().
		Border(1, colors.FieldBorder).Background(colors.Panel)
	if width > 0 {
		row.Width(width)
	}
	draft := ui.Local(row, "number", func() numberDraft {
		return numberDraft{text: format(*value), value: *value}
	})
	if draft.value != *value {
		draft.value, draft.text = *value, format(*value)
	}
	row.Children(func() {
		in := ui.TextInputBase(c.Key("entry"), &draft.text).Grow(1).MinWidth(0).Height(32).
			Padding(0, 10).FontSize(12).Label(label)
		if in.Changed() {
			if n, err := strconv.ParseFloat(draft.text, 64); err == nil && within(n, low, high) {
				*value = math.Round(n*scale) / scale
			}
		}
		delta := 0.0
		switch {
		case in.Shortcut(0, ui.KeyUp):
			delta = step
		case in.Shortcut(0, ui.KeyDown):
			delta = -step
		}
		if delta != 0 {
			*value = min(high, max(low, math.Round((*value+delta)*scale)/scale))
		}
		if unit != "" {
			ui.Text(c, unit).FontSize(10).TextColor(colors.Faint).Margin(0, 4, 0, 0)
		}
		stepped := false
		if stepper {
			stepped = ui.Stepper(c.Key("stepper"), value, low, high, step).Width(20).Height(32).Radius(0).
				BorderWidth(0, 0, 0, 1).BorderColor(colors.FieldBorder).
				Label(name + i18n.Tr("unit.stepper")).Changed()
		}
		if !in.Focused() || stepped || delta != 0 {
			draft.text = format(*value)
		}
		draft.value = *value
	})
}
