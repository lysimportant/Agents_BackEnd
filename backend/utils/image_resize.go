package utils

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// ResizeToFit 将图片等比缩放到不超过 maxWidth × maxHeight 的尺寸，保持宽高比。
// 原图已小于目标尺寸时返回原图，避免小图被放大或二次编码。
func ResizeToFit(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	// scale 保存缩放比例，取宽高两个方向中的较小值。
	scale := 1.0
	if width > maxWidth {
		scale = float64(maxWidth) / float64(width)
	}
	if height > maxHeight {
		if heightScale := float64(maxHeight) / float64(height); heightScale < scale {
			scale = heightScale
		}
	}
	if scale >= 1.0 {
		return src
	}
	// targetWidth、targetHeight 保存缩放后的目标尺寸。
	targetWidth := int(math.Round(float64(width) * scale))
	targetHeight := int(math.Round(float64(height) * scale))
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}
	// 先转成非预乘 NRGBA，避免 alpha 插值出现暗边。
	source := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(source, source.Bounds(), src, bounds.Min, draw.Src)
	// result 保存缩放后的目标图。
	result := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := 0; y < targetHeight; y++ {
		// sy 保存源图对应行坐标（中心对齐）。
		sy := (float64(y)+0.5)/scale - 0.5
		if sy < 0 {
			sy = 0
		}
		y0 := int(sy)
		y1 := y0 + 1
		if y1 >= height {
			y1 = height - 1
		}
		// fy 保存行方向插值权重。
		fy := sy - float64(y0)
		for x := 0; x < targetWidth; x++ {
			sx := (float64(x)+0.5)/scale - 0.5
			if sx < 0 {
				sx = 0
			}
			x0 := int(sx)
			x1 := x0 + 1
			if x1 >= width {
				x1 = width - 1
			}
			// fx 保存列方向插值权重。
			fx := sx - float64(x0)
			// 取四邻域像素做双线性插值。
			c00 := source.NRGBAAt(x0, y0)
			c01 := source.NRGBAAt(x0, y1)
			c10 := source.NRGBAAt(x1, y0)
			c11 := source.NRGBAAt(x1, y1)
			result.SetNRGBA(x, y, color.NRGBA{
				R: interpolateChannel(c00.R, c01.R, c10.R, c11.R, fx, fy),
				G: interpolateChannel(c00.G, c01.G, c10.G, c11.G, fx, fy),
				B: interpolateChannel(c00.B, c01.B, c10.B, c11.B, fx, fy),
				A: interpolateChannel(c00.A, c01.A, c10.A, c11.A, fx, fy),
			})
		}
	}
	return result
}

// interpolateChannel 对单个颜色通道做双线性插值。
func interpolateChannel(v00, v01, v10, v11 uint8, fx, fy float64) uint8 {
	// top、bottom 保存上下两行的水平插值结果。
	top := float64(v00)*(1-fx) + float64(v10)*fx
	bottom := float64(v01)*(1-fx) + float64(v11)*fx
	return uint8(math.Round(top*(1-fy) + bottom*fy))
}
