package main

import "github.com/egoist/mygo/ui"

// openFontLicense 是目录中开源字体采用的许可证
const openFontLicense = "SIL OFL 1.1"

// fontCatalog 保存适合标签的字体、用途和官方发布入口
var fontCatalog = [...]struct {
	name, use, license, url string
}{
	{"思源黑体", "简繁中文 · 商品名、说明小字", openFontLicense, "https://github.com/adobe-fonts/source-han-sans/releases/latest"},
	{"阿里巴巴普惠体", "简体中文 · 商品名、价格和说明", "免费商用", "https://www.alibabafonts.com/#/font"},
	{"Inter", "拉丁字母 · 小字号英文、价格和规格", openFontLicense, "https://rsms.me/inter/download/"},
	{"JetBrains Mono", "拉丁字母 · 等宽批号、SKU，容易区分 0 和 O", openFontLicense, "https://github.com/JetBrains/JetBrainsMono/releases/latest"},
	{"昭源黑体", "繁体中文 · 香港字形、商品说明", openFontLicense, "https://github.com/chiron-fonts/chiron-hei-hk/releases/latest"},
	{"jf open 粉圓", "繁体中文 · 圆体，适合食品和手作标签标题", openFontLicense, "https://github.com/justfont/open-huninn-font/releases/latest"},
	{"思源宋体", "简繁中文 · 品牌名和较大的商品标题", openFontLicense, "https://github.com/adobe-fonts/source-han-serif/releases/latest"},
}

// fontDownloadHeading 显示字体标题和打开下载目录的按钮
func fontDownloadHeading(c *ui.Context) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		sectionHeading(c, "字体")
		ui.Box(c.Key("spacer")).Grow(1)
		button := ui.ButtonBase(c.Key("font-download")).Role(ui.RoleButton).Cursor(ui.CursorPointer)
		button.Children(func() {
			text := ui.Text(c, "下载开源字体").FontSize(10.5).TextColor(ui.Hex(colorAccent))
			if button.Hovered() {
				text.Underline()
			}
		})
		open := ui.Local(button, "open", func() bool { return false })
		if button.Clicked() {
			*open = true
		}
		fontCatalogDialog(c.Key("font-catalog"), open)
	})
}

// fontCatalogDialog 显示字体用途、授权和官方下载链接
func fontCatalogDialog(c *ui.Context, open *bool) {
	t := c.Theme()
	ui.Modal(c, open, func() {
		ui.Column(c).Width(490).Gap(12).Children(func() {
			ui.Row(c.Key("heading")).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "下载开源字体").FontSize(16).Bold().Grow(1)
				if ui.Button(c.Key("close"), "关闭").Height(26).FontSize(12).Clicked() {
					*open = false
				}
			})
			ui.Column(c.Key("fonts")).Gap(6).Children(func() {
				for _, font := range fontCatalog {
					row := c.Key(font.name)
					ui.Row(row).Padding(8, 12).Gap(12).AlignItems(ui.Center).Radius(7).Background(t.Surface).Children(func() {
						ui.Column(row).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(row, font.name).FontSize(13).Bold()
							ui.Text(row, font.use).FontSize(11).TextColor(t.TextMuted)
						})
						ui.Text(row, font.license).Width(72).FontSize(11).TextColor(t.TextMuted)
						ui.Link(row.Key("download"), "官方下载", font.url).FontSize(12).Label("下载 " + font.name)
					})
				}
			})
			ui.Text(c, "安装字体后，重新打开 GoLabel 即可使用。").FontSize(11).TextColor(t.TextMuted)
		})
	}).Label("下载开源字体")
}
