package main

import "github.com/egoist/mygo/ui"

// openFontLicense 是目录中开源字体采用的许可证
const openFontLicense = "SIL OFL 1.1"

// fontCatalog 保存适合标签的字体、用途和官方发布入口，名称和用途随界面语言显示
var fontCatalog = [...]struct {
	name, use    text
	license, url string
}{
	{text{"思源黑体", "Source Han Sans"}, text{"简繁中文 · 商品名、说明小字", "Simplified and traditional Chinese · product names, fine print"}, openFontLicense, "https://github.com/adobe-fonts/source-han-sans/releases/latest"},
	{text{"阿里巴巴普惠体", "Alibaba PuHuiTi"}, text{"简体中文 · 商品名、价格和说明", "Simplified Chinese · product names, prices and descriptions"}, openFontLicense, "https://www.alibabafonts.com/#/font"},
	{text{"Inter", "Inter"}, text{"拉丁字母 · 小字号英文、价格和规格", "Latin · small English text, prices and specs"}, openFontLicense, "https://rsms.me/inter/download/"},
	{text{"JetBrains Mono", "JetBrains Mono"}, text{"拉丁字母 · 等宽批号、SKU，容易区分 0 和 O", "Latin · monospaced batch numbers and SKUs; 0 and O stay distinct"}, openFontLicense, "https://github.com/JetBrains/JetBrainsMono/releases/latest"},
	{text{"昭源黑体", "Chiron Hei HK"}, text{"繁体中文 · 香港字形、商品说明", "Traditional Chinese · Hong Kong glyphs, product descriptions"}, openFontLicense, "https://github.com/chiron-fonts/chiron-hei-hk/releases/latest"},
	{text{"jf open 粉圓", "jf open huninn"}, text{"繁体中文 · 圆体，适合食品和手作标签标题", "Traditional Chinese · rounded, for food and handmade label titles"}, openFontLicense, "https://github.com/justfont/open-huninn-font/releases/latest"},
	{text{"思源宋体", "Source Han Serif"}, text{"简繁中文 · 品牌名和较大的商品标题", "Simplified and traditional Chinese · brand names and larger headings"}, openFontLicense, "https://github.com/adobe-fonts/source-han-serif/releases/latest"},
}

// fontDownloadHeading 显示字体标题和打开下载目录的按钮
func fontDownloadHeading(c *ui.Context) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		sectionHeading(c, tr("font.title"))
		ui.Box(c.Key("spacer")).Grow(1)
		button := ui.ButtonBase(c.Key("font-download")).Role(ui.RoleButton).Cursor(ui.CursorPointer)
		button.Children(func() {
			text := ui.Text(c, tr("font.download")).FontSize(10.5).TextColor(colors.Accent)
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
				ui.Text(c, tr("font.download")).FontSize(16).Bold().Grow(1)
				if ui.Button(c.Key("close"), tr("font.close")).Height(26).FontSize(12).Clicked() {
					*open = false
				}
			})
			ui.Column(c.Key("fonts")).Gap(6).Children(func() {
				for _, font := range fontCatalog {
					name := font.name.get()
					row := c.Key(name)
					ui.Row(row).Padding(8, 12).Gap(12).AlignItems(ui.Center).Radius(7).Background(t.Surface).Children(func() {
						ui.Column(row).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(row, name).FontSize(13).Bold()
							ui.Text(row, font.use.get()).FontSize(11).TextColor(t.TextMuted)
						})
						ui.Text(row, font.license).Width(72).FontSize(11).TextColor(t.TextMuted)
						ui.Link(row.Key("download"), tr("font.official"), font.url).FontSize(12).
							Label(trf("font.downloadName", name))
					})
				}
			})
			ui.Text(c, tr("font.restart")).FontSize(11).TextColor(t.TextMuted)
		})
	}).Label(tr("font.download"))
}
