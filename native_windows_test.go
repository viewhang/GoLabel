package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestWindowsLabelPDF 验证包含条码和二维码的实际尺寸 PDF 可以覆盖已有文件
func TestWindowsLabelPDF(t *testing.T) {
	g := &labelGraphics{}
	l := newLabel()
	l.Items = []item{
		{Kind: kindBarcode, Value: "12345678", X: 3, Y: 2, W: 38, H: 9},
		{Kind: kindQR, Value: "GoLabel-QR", X: 3, Y: 12, W: 16, H: 16},
	}
	for i := range l.Items {
		l.Items[i].prepareSymbol(g)
	}
	pdf, err := g.pdf(l)
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
	if err := writePDFFile(path, pdf); err != nil {
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
