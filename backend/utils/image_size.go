// Package utils 提供图片尺寸探测等通用工具能力。
package utils

import (
	"encoding/binary"
	"io"
)

// ImageDimensions 表示图片的宽高尺寸。
type ImageDimensions struct {
	// Width 图片宽度（像素）。
	Width int
	// Height 图片高度（像素）。
	Height int
}

// DetectImageDimensions 探测常见图片格式 PNG、JPEG、GIF 与 WebP 的尺寸。
func DetectImageDimensions(r io.Reader) (ImageDimensions, bool) {
	// header 保存用于探测的文件头部数据。
	header := make([]byte, 512)
	// n 保存实际读取到的字节数。
	n, err := io.ReadFull(r, header)
	if err != nil && n < 4 {
		return ImageDimensions{}, false
	}
	// data 保存实际读取到的头部切片。
	data := header[:n]
	// 是否为 PNG 格式。
	if len(data) >= 24 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ImageDimensions{
			Width:  int(binary.BigEndian.Uint32(data[16:20])),
			Height: int(binary.BigEndian.Uint32(data[20:24])),
		}, true
	}
	// 是否为 GIF 格式。
	if len(data) >= 10 && string(data[:3]) == "GIF" {
		return ImageDimensions{
			Width:  int(binary.LittleEndian.Uint16(data[6:8])),
			Height: int(binary.LittleEndian.Uint16(data[8:10])),
		}, true
	}
	// 是否为 JPEG 格式。
	if len(data) >= 4 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return detectJPEGDimensions(data)
	}
	// 是否为 WebP 格式。
	if len(data) >= 30 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		// 按子格式 VP8 / VP8L / VP8X 分别解析尺寸。
		// VP8 lossy 有损格式。
		if len(data) >= 26 && string(data[12:16]) == "VP8 " {
			return ImageDimensions{
				Width:  int(binary.LittleEndian.Uint16(data[26:28]) & 0x3FFF),
				Height: int(binary.LittleEndian.Uint16(data[28:30]) & 0x3FFF),
			}, true
		}
		// VP8L lossless 无损格式。
		if len(data) >= 25 && string(data[12:16]) == "VP8L" {
			// b0、b1、b2、b3 保存位流中的原始字节。
			b0, b1, b2, b3 := data[21], data[22], data[23], data[24]
			// widthBits 保存宽度位，取低 14 位拼接。
			widthBits := uint32(b1&0x3F)<<8 | uint32(b0)
			// heightBits 保存高度位，取低 14 位拼接。
			heightBits := uint32(b3)<<6 | uint32(b2&0x3F)
			return ImageDimensions{
				Width:  int(widthBits) + 1,
				Height: int(heightBits) + 1,
			}, true
		}
		// VP8X extended 扩展格式。
		if len(data) >= 30 && string(data[12:16]) == "VP8X" {
			return ImageDimensions{
				Width:  int(readUint24LE(data[24:27])) + 1,
				Height: int(readUint24LE(data[27:30])) + 1,
			}, true
		}
	}
	return ImageDimensions{}, false
}

// detectJPEGDimensions 解析 JPEG 头中的 SOF 段获取尺寸。
func detectJPEGDimensions(data []byte) (ImageDimensions, bool) {
	// offset 保存当前解析的偏移位置。
	offset := 2
	for offset+9 < len(data) {
		if data[offset] != 0xFF {
			offset++
			continue
		}
		// marker 保存当前段标记。
		marker := data[offset+1]
		// 跳过无数据段与重启标记。
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			offset += 2
			continue
		}
		// segmentLength 保存当前段长度。
		segmentLength := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		if segmentLength < 2 {
			return ImageDimensions{}, false
		}
		// 命中 SOF0-SOF15 帧段时读取宽高。
		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			if offset+9 < len(data) {
				return ImageDimensions{
					Height: int(binary.BigEndian.Uint16(data[offset+5 : offset+7])),
					Width:  int(binary.BigEndian.Uint16(data[offset+7 : offset+9])),
				}, true
			}
			return ImageDimensions{}, false
		}
		offset += 2 + segmentLength
	}
	return ImageDimensions{}, false
}

// readUint24LE 读取小端序的 24 位无符号整数。
func readUint24LE(data []byte) uint32 {
	if len(data) < 3 {
		return 0
	}
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
}