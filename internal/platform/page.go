package platform

import "image"

// Page 保存待导出或打印的标签位图及纸张尺寸
type Page struct {
	Raster            *image.RGBA
	WidthMM, HeightMM float64
}

// SizePoints 把纸张的毫米宽高换算为 PDF 点
func (p Page) SizePoints() (width, height float64) {
	return p.WidthMM * 72 / 25.4, p.HeightMM * 72 / 25.4
}
