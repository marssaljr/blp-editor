package blp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
)

// DecodeMipmap decodes a specific mipmap level to an image.
func (blp *BLP) DecodeMipmap(level int) (image.Image, error) {
	data, err := blp.GetMipmapData(level)
	if err != nil {
		return nil, err
	}

	width, height, err := blp.GetMipmapDimensions(level)
	if err != nil {
		return nil, err
	}

	switch blp.Format {
	case FormatPaletted:
		return decodePaletted(blp, width, height, data)
	case FormatDXT1, FormatBC1:
		return decodeDXT1(width, height, data, blp.HasAlpha)
	case FormatDXT3, FormatBC2:
		return decodeDXT3(width, height, data)
	case FormatDXT5, FormatBC3:
		return decodeDXT5(width, height, data)
	case FormatARGB8888:
		return decodeBGRA(width, height, data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", blp.FormatName())
	}
}

// decodePaletted decodes the paletted BLP format (indexed RGB plus packed alpha).
func decodePaletted(blp *BLP, width, height uint32, data []byte) (image.Image, error) {
	pixelCount := int(width * height)
	if len(data) < pixelCount {
		return nil, errors.New("paletted data too short")
	}

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	alphaData := data[pixelCount:]

	for i := 0; i < pixelCount; i++ {
		x := i % int(width)
		y := i / int(width)
		index := int(data[i])
		if index > 255 {
			index = 0
		}
		c := blp.Palette[index]
		c.A = palettedAlpha(alphaData, i, blp.AlphaBitDepth)
		img.Set(x, y, c)
	}

	return img, nil
}

func palettedAlpha(alphaData []byte, pixelIndex int, bitDepth uint8) uint8 {
	switch bitDepth {
	case 1:
		byteIndex := pixelIndex / 8
		if byteIndex >= len(alphaData) {
			return 255
		}
		if alphaData[byteIndex]&(1<<uint(pixelIndex%8)) != 0 {
			return 255
		}
		return 0
	case 4:
		byteIndex := pixelIndex / 2
		if byteIndex >= len(alphaData) {
			return 255
		}
		var a uint8
		if pixelIndex%2 == 0 {
			a = alphaData[byteIndex] & 0x0F
		} else {
			a = alphaData[byteIndex] >> 4
		}
		return a * 17
	case 8:
		if pixelIndex >= len(alphaData) {
			return 255
		}
		return alphaData[pixelIndex]
	default:
		return 255
	}
}

// decodeBGRA decodes uncompressed 32-bit BGRA pixels.
func decodeBGRA(width, height uint32, data []byte) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	pixelCount := int(width * height)
	for i := 0; i < pixelCount && i*4+3 < len(data); i++ {
		offset := i * 4
		x := i % int(width)
		y := i / int(width)
		img.Set(x, y, color.RGBA{R: data[offset+2], G: data[offset+1], B: data[offset], A: data[offset+3]})
	}

	return img, nil
}

// decodeDXT1 decodes DXT1/BC1 compressed data.
func decodeDXT1(width, height uint32, data []byte, hasAlpha bool) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	blockSize := 8
	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4

	for by := uint32(0); by < blocksY; by++ {
		for bx := uint32(0); bx < blocksX; bx++ {
			blockOffset := int(by*blocksX+bx) * blockSize
			if blockOffset+blockSize > len(data) {
				return img, nil
			}
			decodeDXT1Block(img, data[blockOffset:blockOffset+blockSize], bx*4, by*4, width, height, hasAlpha)
		}
	}

	return img, nil
}

// decodeDXT3 decodes DXT3/BC2 compressed data.
func decodeDXT3(width, height uint32, data []byte) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	blockSize := 16
	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4

	for by := uint32(0); by < blocksY; by++ {
		for bx := uint32(0); bx < blocksX; bx++ {
			blockOffset := int(by*blocksX+bx) * blockSize
			if blockOffset+blockSize > len(data) {
				return img, nil
			}
			decodeDXT3Block(img, data[blockOffset:blockOffset+blockSize], bx*4, by*4, width, height)
		}
	}

	return img, nil
}

// decodeDXT5 decodes DXT5/BC3 compressed data.
func decodeDXT5(width, height uint32, data []byte) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

	blockSize := 16
	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4

	for by := uint32(0); by < blocksY; by++ {
		for bx := uint32(0); bx < blocksX; bx++ {
			blockOffset := int(by*blocksX+bx) * blockSize
			if blockOffset+blockSize > len(data) {
				return img, nil
			}
			decodeDXT5Block(img, data[blockOffset:blockOffset+blockSize], bx*4, by*4, width, height)
		}
	}

	return img, nil
}

func decodeDXT1Block(img *image.RGBA, block []byte, bx, by, width, height uint32, hasAlpha bool) {
	c0 := binary.LittleEndian.Uint16(block[0:2])
	c1 := binary.LittleEndian.Uint16(block[2:4])

	colors := [4]color.RGBA{rgb565ToRGBA(c0), rgb565ToRGBA(c1), {}, {}}
	if c0 > c1 {
		colors[2] = interpolateRGB(colors[0], colors[1], 2, 1, 3)
		colors[3] = interpolateRGB(colors[0], colors[1], 1, 2, 3)
	} else {
		colors[2] = interpolateRGB(colors[0], colors[1], 1, 1, 2)
		if hasAlpha {
			colors[3] = color.RGBA{0, 0, 0, 0}
		} else {
			colors[3] = color.RGBA{0, 0, 0, 255}
		}
	}

	indices := block[4:8]
	for y := uint32(0); y < 4; y++ {
		for x := uint32(0); x < 4; x++ {
			if bx+x >= width || by+y >= height {
				continue
			}
			pixelIndex := y*4 + x
			byteIndex := pixelIndex / 4
			bit := uint((pixelIndex % 4) * 2)
			idx := (indices[byteIndex] >> bit) & 0x03
			img.Set(int(bx+x), int(by+y), colors[idx])
		}
	}
}

func decodeDXT3Block(img *image.RGBA, block []byte, bx, by, width, height uint32) {
	c0 := binary.LittleEndian.Uint16(block[8:10])
	c1 := binary.LittleEndian.Uint16(block[10:12])

	colors := [4]color.RGBA{
		rgb565ToRGBA(c0),
		rgb565ToRGBA(c1),
		interpolateRGB(rgb565ToRGBA(c0), rgb565ToRGBA(c1), 2, 1, 3),
		interpolateRGB(rgb565ToRGBA(c0), rgb565ToRGBA(c1), 1, 2, 3),
	}

	indices := block[12:16]
	for y := uint32(0); y < 4; y++ {
		for x := uint32(0); x < 4; x++ {
			if bx+x >= width || by+y >= height {
				continue
			}
			pixelIndex := y*4 + x

			byteIndex := pixelIndex / 4
			bit := uint((pixelIndex % 4) * 2)
			c := colors[(indices[byteIndex]>>bit)&0x03]

			alphaByteIndex := pixelIndex / 2
			alphaShift := uint((pixelIndex % 2) * 4)
			c.A = ((block[alphaByteIndex] >> alphaShift) & 0x0F) * 17

			img.Set(int(bx+x), int(by+y), c)
		}
	}
}

func decodeDXT5Block(img *image.RGBA, block []byte, bx, by, width, height uint32) {
	a0 := block[0]
	a1 := block[1]
	alphaIndices := block[2:8]

	c0 := binary.LittleEndian.Uint16(block[8:10])
	c1 := binary.LittleEndian.Uint16(block[10:12])
	colors := [4]color.RGBA{
		rgb565ToRGBA(c0),
		rgb565ToRGBA(c1),
		interpolateRGB(rgb565ToRGBA(c0), rgb565ToRGBA(c1), 2, 1, 3),
		interpolateRGB(rgb565ToRGBA(c0), rgb565ToRGBA(c1), 1, 2, 3),
	}

	colorIndices := block[12:16]
	for y := uint32(0); y < 4; y++ {
		for x := uint32(0); x < 4; x++ {
			if bx+x >= width || by+y >= height {
				continue
			}
			pixelIndex := y*4 + x

			byteIndex := pixelIndex / 4
			bit := uint((pixelIndex % 4) * 2)
			c := colors[(colorIndices[byteIndex]>>bit)&0x03]
			c.A = decodeDXT5AlphaPixel(a0, a1, alphaIndices, int(pixelIndex))

			img.Set(int(bx+x), int(by+y), c)
		}
	}
}

func decodeDXT5AlphaPixel(a0, a1 uint8, indices []byte, pixelIndex int) uint8 {
	bitOffset := pixelIndex * 3
	byteIndex := bitOffset / 8
	shift := uint(bitOffset % 8)
	if byteIndex >= len(indices) {
		return 255
	}

	idx := uint16(indices[byteIndex]>>shift) & 0x07
	if shift > 5 && byteIndex+1 < len(indices) {
		idx |= (uint16(indices[byteIndex+1]) << (8 - shift)) & 0x07
	}

	var alpha [8]uint16
	if a0 > a1 {
		alpha[0] = uint16(a0)
		alpha[1] = uint16(a1)
		for i := 2; i < 8; i++ {
			alpha[i] = (uint16(8-i)*uint16(a0) + uint16(i-1)*uint16(a1)) / 7
		}
	} else {
		alpha[0] = uint16(a0)
		alpha[1] = uint16(a1)
		for i := 2; i < 6; i++ {
			alpha[i] = (uint16(6-i)*uint16(a0) + uint16(i-1)*uint16(a1)) / 5
		}
		alpha[6] = 0
		alpha[7] = 255
	}

	return uint8(alpha[idx])
}

func interpolateRGB(c0, c1 color.RGBA, w0, w1, denom uint32) color.RGBA {
	return color.RGBA{
		R: uint8((w0*uint32(c0.R) + w1*uint32(c1.R)) / denom),
		G: uint8((w0*uint32(c0.G) + w1*uint32(c1.G)) / denom),
		B: uint8((w0*uint32(c0.B) + w1*uint32(c1.B)) / denom),
		A: 255,
	}
}

// rgb565ToRGBA converts an RGB565 value to RGBA using bit replication.
func rgb565ToRGBA(v uint16) color.RGBA {
	r5 := uint8((v >> 11) & 0x1F)
	g6 := uint8((v >> 5) & 0x3F)
	b5 := uint8(v & 0x1F)

	return color.RGBA{
		R: uint8(r5<<3 | r5>>2),
		G: uint8(g6<<2 | g6>>4),
		B: uint8(b5<<3 | b5>>2),
		A: 255,
	}
}

// rgbaToRGB565 converts RGB values to an RGB565 value.
func rgbaToRGB565(r, g, b uint8) uint16 {
	r5 := uint16((uint32(r)*31 + 127) / 255)
	g6 := uint16((uint32(g)*63 + 127) / 255)
	b5 := uint16((uint32(b)*31 + 127) / 255)
	return r5<<11 | g6<<5 | b5
}

// CalculateNextMipmapDimensions calculates the dimensions for the next mipmap level.
func CalculateNextMipmapDimensions(width, height uint32) (uint32, uint32) {
	nextWidth := width / 2
	nextHeight := height / 2
	if nextWidth < 1 {
		nextWidth = 1
	}
	if nextHeight < 1 {
		nextHeight = 1
	}
	return nextWidth, nextHeight
}

// CalculateMipmapLevels calculates the number of mipmap levels needed.
func CalculateMipmapLevels(width, height uint32) uint32 {
	maxDim := width
	if height > maxDim {
		maxDim = height
	}
	return uint32(math.Log2(float64(maxDim))) + 1
}
