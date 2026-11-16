package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pengh/GoLabel/internal/platform"
)

// TestWindowsFontNames 验证系统字体的中文名称、英文切换和原始字体族映射
func TestWindowsFontNames(t *testing.T) {
	resetLanguage(t)
	setAppLocale(string(languageZhHans))
	chinese := makeFontMenu()
	if got := platform.FontFamilyName("Arial"); got != "Arial" {
		t.Fatalf("字体原名未保留：%q", got)
	}
	setAppLocale(string(languageEnglish))
	english := makeFontMenu()
	tested := 0
	for _, font := range []struct{ family, name string }{
		{"Microsoft YaHei", "微软雅黑"},
		{"SimSun", "宋体"},
		{"SimHei", "黑体"},
	} {
		if _, installed := chinese.nameByFamily[font.family]; !installed {
			continue
		}
		tested++
		if name := chinese.nameByFamily[font.family]; name != font.name {
			t.Errorf("字体 %q 未显示中文名称：%q", font.family, name)
		}
		if chinese.familyByName[font.name] != font.family || english.nameByFamily[font.family] != font.family {
			t.Errorf("字体 %q 的原始族名映射或英文切换不正确", font.family)
		}
	}
	if tested == 0 {
		t.Skip("系统未安装微软雅黑、宋体或黑体")
	}
}

// TestWindowsLabelPDF 验证包含条码和二维码的实际尺寸 PDF 可以覆盖已有文件
func TestWindowsLabelPDF(t *testing.T) {
	g := &platform.Graphics{}
	l := newLabel()
	l.Items = []item{
		{Kind: kindBarcode, Value: "12345678", X: 3, Y: 2, W: 38, H: 9},
		{Kind: kindQR, Value: "GoLabel-QR", X: 3, Y: 12, W: 16, H: 16},
	}
	for i := range l.Items {
		l.Items[i].prepareSymbol(g)
	}
	page, err := l.printPage()
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := g.PDF(page)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(pdf, []byte("/MediaBox [0 0 141.73 85.04]")) {
		t.Fatal("PDF 页面不是 50 × 30 毫米")
	}
	path := filepath.Join(t.TempDir(), "标签.pdf")
	if err := os.WriteFile(path, []byte("旧文件"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := platform.WritePDFFile(path, pdf); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, pdf) {
		t.Fatal("导出的 PDF 未完整替换旧文件")
	}
}
