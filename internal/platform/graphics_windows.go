package platform

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"codeberg.org/go-pdf/fpdf"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
)

// Graphics 提供 Windows 标签的条码和 PDF 操作
type Graphics struct{}

// OpenGraphics 创建 Windows 标签图形后端
func OpenGraphics() (*Graphics, error) { return &Graphics{}, nil }

// FontFamilyName 返回 DirectWrite 提供的字体族名
func FontFamilyName(family string) string { return family }

// SymbolImage 生成原始码模块并为 Code 128 保留两侧静区
func (g *Graphics) SymbolImage(code128Barcode bool, value string) (image.Image, error) {
	if !code128Barcode {
		return qr.Encode(value, qr.M, qr.Auto)
	}
	code, err := code128.Encode(value)
	if err != nil {
		return nil, err
	}
	// quietZone 是 Code 128 每侧白边的模块数
	const quietZone = 10
	bounds := code.Bounds()
	padded := bounds
	padded.Min.X -= quietZone
	padded.Max.X += quietZone
	bitmap := image.NewGray(padded)
	draw.Draw(bitmap, bitmap.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(bitmap, bounds, code, bounds.Min, draw.Src)
	return bitmap, nil
}

// PDF 生成与标签实际宽高相同的单页 PDF
func (g *Graphics) PDF(page Page) ([]byte, error) {
	var pngData, pdfData bytes.Buffer
	if err := png.Encode(&pngData, page.Raster); err != nil {
		return nil, err
	}
	doc := fpdf.NewCustom(&fpdf.InitType{UnitStr: "mm", Size: fpdf.SizeType{Wd: page.WidthMM, Ht: page.HeightMM}})
	doc.AddPage()
	options := fpdf.ImageOptions{ImageType: "PNG"}
	doc.RegisterImageOptionsReader("label", options, &pngData)
	doc.ImageOptions("label", 0, 0, page.WidthMM, page.HeightMM, false, options, 0, "")
	if err := doc.Output(&pdfData); err != nil {
		return nil, err
	}
	return pdfData.Bytes(), nil
}

// WritePDFFile 在写完临时文件后替换目标 PDF
func WritePDFFile(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".golabel-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
