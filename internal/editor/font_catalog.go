package editor

import (
	"github.com/egoist/mygo/ui"
	"github.com/pengh/GoLabel/internal/i18n"
)

// openFontLicense 是目录中开源字体采用的许可证
var openFontLicense = i18n.Text{ZH: "SIL OFL 1.1", EN: "SIL OFL 1.1"}

// fontCatalog 保存适合标签的字体、用途和官方发布入口，名称和用途随界面语言显示
var fontCatalog = [...]struct {
	name, use, license i18n.Text
	url                string
}{
	{i18n.Text{ZH: "思源黑体", EN: "Source Han Sans"}, i18n.Text{ZH: "简繁中文 · 商品名、说明小字", EN: "Simplified and traditional Chinese · product names, fine print"}, openFontLicense, "https://github.com/adobe-fonts/source-han-sans/releases/latest"},
	{i18n.Text{ZH: "阿里巴巴普惠体", EN: "Alibaba PuHuiTi"}, i18n.Text{ZH: "简体中文 · 商品名、价格和说明", EN: "Simplified Chinese · product names, prices and descriptions"}, i18n.Text{ZH: "免费商用", EN: "Free for commercial use"}, "https://www.alibabafonts.com/#/font"},
	{i18n.Text{ZH: "Inter", EN: "Inter"}, i18n.Text{ZH: "拉丁字母 · 小字号英文、价格和规格", EN: "Latin · small English text, prices and specs"}, openFontLicense, "https://rsms.me/inter/download/"},
	{i18n.Text{ZH: "JetBrains Mono", EN: "JetBrains Mono"}, i18n.Text{ZH: "拉丁字母 · 等宽批号、SKU，容易区分 0 和 O", EN: "Latin · monospaced batch numbers and SKUs; 0 and O stay distinct"}, openFontLicense, "https://github.com/JetBrains/JetBrainsMono/releases/latest"},
	{i18n.Text{ZH: "昭源黑体", EN: "Chiron Hei HK"}, i18n.Text{ZH: "繁体中文 · 香港字形、商品说明", EN: "Traditional Chinese · Hong Kong glyphs, product descriptions"}, openFontLicense, "https://github.com/chiron-fonts/chiron-hei-hk/releases/latest"},
	{i18n.Text{ZH: "jf open 粉圓", EN: "jf open huninn"}, i18n.Text{ZH: "繁体中文 · 圆体，适合食品和手作标签标题", EN: "Traditional Chinese · rounded, for food and handmade label titles"}, openFontLicense, "https://github.com/justfont/open-huninn-font/releases/latest"},
	{i18n.Text{ZH: "思源宋体", EN: "Source Han Serif"}, i18n.Text{ZH: "简繁中文 · 品牌名和较大的商品标题", EN: "Simplified and traditional Chinese · brand names and larger headings"}, openFontLicense, "https://github.com/adobe-fonts/source-han-serif/releases/latest"},
}

// fontDownloadHeading 显示字体标题和打开下载目录的按钮
func fontDownloadHeading(c *ui.Context) {
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		sectionHeading(c, i18n.Tr("font.title"))
		ui.Box(c.Key("spacer")).Grow(1)
		button := ui.ButtonBase(c.Key("font-download")).Role(ui.RoleButton).Cursor(ui.CursorPointer)
		button.Children(func() {
			text := ui.Text(c, i18n.Tr("font.download")).FontSize(10.5).TextColor(colors.Accent)
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
				ui.Text(c, i18n.Tr("font.download")).FontSize(16).Bold().Grow(1)
				if ui.Button(c.Key("close"), i18n.Tr("font.close")).Height(26).FontSize(12).Clicked() {
					*open = false
				}
			})
			ui.Column(c.Key("fonts")).Gap(6).Children(func() {
				for _, font := range fontCatalog {
					name := font.name.Get()
					row := c.Key(name)
					ui.Row(row).Padding(8, 12).Gap(12).AlignItems(ui.Center).Radius(7).Background(t.Surface).Children(func() {
						ui.Column(row).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(row, name).FontSize(13).Bold()
							ui.Text(row, font.use.Get()).FontSize(11).TextColor(t.TextMuted)
						})
						ui.Text(row, font.license.Get()).Width(72).FontSize(11).TextColor(t.TextMuted)
						ui.Link(row.Key("download"), i18n.Tr("font.official"), font.url).FontSize(12).
							Label(i18n.Trf("font.downloadName", name))
					})
				}
			})
			ui.Text(c, i18n.Tr("font.restart")).FontSize(11).TextColor(t.TextMuted)
		})
	}).Label(i18n.Tr("font.download"))
}
