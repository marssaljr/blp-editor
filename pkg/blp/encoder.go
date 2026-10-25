package blp

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"math"
)

// ErrUnsupportedFormat is returned when an image cannot be encoded as the
// requested BLP format.
var ErrUnsupportedFormat = errors.New("unsupported BLP format")

// EncodeFromImage encodes an image to the requested logical BLP format and
// stores the result as the base mipmap level.
func (blp *BLP) EncodeFromImage(img image.Image, format uint32) error {
	bounds := img.Bounds()
	blp.Width = uint32(bounds.Dx())
	blp.Height = uint32(bounds.Dy())
	blp.Format = format
	blp.ColorEncoding, blp.AlphaBitDepth, blp.AlphaType = diskFieldsFromFormat(format)

	if format == FormatPaletted {
		blp.AlphaBitDepth = 0
		if imageHasAlpha(img) {
			blp.AlphaBitDepth = 8
		}
	} else {
		blp.Palette = [256]color.RGBA{}
	}

	var (
		data []byte
		err  error
	)
	switch format {
	case FormatPaletted:
		data, blp.Palette, err = encodePaletted(img, blp.AlphaBitDepth)
	case FormatDXT1, FormatBC1:
		data, err = encodeDXT1(img)
	case FormatDXT3, FormatBC2:
		data, err = encodeDXT3(img)
	case FormatDXT5, FormatBC3:
		data, err = encodeDXT5(img)
	case FormatARGB8888:
		data, err = encodeBGRA(img)
	default:
		return ErrUnsupportedFormat
	}
	if err != nil {
		return err
	}

	blp.HasAlpha = blp.AlphaBitDepth > 0 || format == FormatDXT3 || format == FormatDXT5 || format == FormatARGB8888
	blp.Data = data
	blp.MipmapCount = 1
	blp.MipmapOffsets = make([]uint32, 16)
	blp.MipmapSizes = make([]uint32, 16)
	blp.MipmapOffsets[0] = 0
	blp.MipmapSizes[0] = uint32(len(data))
	blp.HasMipmaps = 0
	return nil
}

func imageHasAlpha(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xFFFF {
				return true
			}
		}
	}
	return false
}

func encodePaletted(img image.Image, alphaBitDepth uint8) ([]byte, [256]color.RGBA, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	pixelCount := width * height

	palette := generatePalette()

	alphaSize := 0
	if alphaBitDepth == 8 {
		alphaSize = pixelCount
	}
	data := make([]byte, pixelCount+alphaSize)

	for i := 0; i < pixelCount; i++ {
		x := i % width
		y := i / width
		c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
		data[i] = byte(findClosestPaletteIndex(palette, c))
		if alphaBitDepth == 8 {
			data[pixelCount+i] = c.A
		}
	}

	return data, palette, nil
}

func encodeBGRA(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	data := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			offset := (y*width + x) * 4
			data[offset] = c.B
			data[offset+1] = c.G
			data[offset+2] = c.R
			data[offset+3] = c.A
		}
	}
	return data, nil
}

func encodeDXT1(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4
	blockSize := 8

	data := make([]byte, blocksX*blocksY*blockSize)
	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			offset := (by*blocksX + bx) * blockSize
			blockImg := extractBlock(img, bx*4, by*4, width, height)
			encodeDXT1Block(data[offset:offset+blockSize], blockImg)
		}
	}
	return data, nil
}

func encodeDXT3(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4
	blockSize := 16

	data := make([]byte, blocksX*blocksY*blockSize)
	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			offset := (by*blocksX + bx) * blockSize
			blockImg := extractBlock(img, bx*4, by*4, width, height)
			encodeDXT3Block(data[offset:offset+blockSize], blockImg)
		}
	}
	return data, nil
}

func encodeDXT5(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	blocksX := (width + 3) / 4
	blocksY := (height + 3) / 4
	blockSize := 16

	data := make([]byte, blocksX*blocksY*blockSize)
	for by := 0; by < blocksY; by++ {
		for bx := 0; bx < blocksX; bx++ {
			offset := (by*blocksX + bx) * blockSize
			blockImg := extractBlock(img, bx*4, by*4, width, height)
			encodeDXT5Block(data[offset:offset+blockSize], blockImg)
		}
	}
	return data, nil
}

func extractBlock(img image.Image, startX, startY, imgWidth, imgHeight int) *image.RGBA {
	block := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			imgX := startX + x
			imgY := startY + y
			if imgX < imgWidth && imgY < imgHeight {
				block.Set(x, y, color.NRGBAModel.Convert(img.At(imgX, imgY)))
			} else {
				block.Set(x, y, color.NRGBA{0, 0, 0, 0})
			}
		}
	}
	return block
}

func blockColors(img *image.RGBA) [16]color.RGBA {
	var colors [16]color.RGBA
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			colors[y*4+x] = img.RGBAAt(x, y)
		}
	}
	return colors
}

func encodeDXT1Block(block []byte, img *image.RGBA) {
	colors := blockColors(img)
	c0, c1 := selectEndpoints(colors)

	binary.LittleEndian.PutUint16(block[0:2], rgbaToRGB565(c0.R, c0.G, c0.B))
	binary.LittleEndian.PutUint16(block[2:4], rgbaToRGB565(c1.R, c1.G, c1.B))

	indices := computeColorIndices(colors, c0, c1, false)
	copy(block[4:8], indices)
}

func encodeDXT3Block(block []byte, img *image.RGBA) {
	// Encode explicit 4-bit alpha for each pixel into block[0:8].
	for i := 0; i < 16; i++ {
		a := img.RGBAAt(i%4, i/4).A
		nibble := uint16((uint16(a) + 8) / 17)
		byteIndex := i / 2
		if i%2 == 0 {
			block[byteIndex] |= byte(nibble & 0x0F)
		} else {
			block[byteIndex] |= byte((nibble & 0x0F) << 4)
		}
	}

	colors := blockColors(img)
	c0, c1 := selectEndpoints(colors)

	binary.LittleEndian.PutUint16(block[8:10], rgbaToRGB565(c0.R, c0.G, c0.B))
	binary.LittleEndian.PutUint16(block[10:12], rgbaToRGB565(c1.R, c1.G, c1.B))

	indices := computeColorIndices(colors, c0, c1, true)
	copy(block[12:16], indices)
}

func encodeDXT5Block(block []byte, img *image.RGBA) {
	colors := blockColors(img)
	encodeDXT5Alpha(block[0:8], colors)

	c0, c1 := selectEndpoints(colors)
	binary.LittleEndian.PutUint16(block[8:10], rgbaToRGB565(c0.R, c0.G, c0.B))
	binary.LittleEndian.PutUint16(block[10:12], rgbaToRGB565(c1.R, c1.G, c1.B))

	indices := computeColorIndices(colors, c0, c1, true)
	copy(block[12:16], indices)
}

func encodeDXT5Alpha(block []byte, colors [16]color.RGBA) {
	var minA, maxA uint8 = 255, 0
	for _, c := range colors {
		if c.A < minA {
			minA = c.A
		}
		if c.A > maxA {
			maxA = c.A
		}
	}

	a0 := maxA
	a1 := minA
	block[0] = a0
	block[1] = a1

	var alpha [8]uint8
	if a0 > a1 {
		alpha[0] = a0
		alpha[1] = a1
		for i := 2; i < 8; i++ {
			alpha[i] = uint8((uint16(8-i)*uint16(a0) + uint16(i-1)*uint16(a1)) / 7)
		}
	} else {
		alpha[0] = a0
		alpha[1] = a1
		for i := 2; i < 6; i++ {
			alpha[i] = uint8((uint16(6-i)*uint16(a0) + uint16(i-1)*uint16(a1)) / 5)
		}
		alpha[6] = 0
		alpha[7] = 255
	}

	for i := 0; i < 16; i++ {
		target := uint16(colors[i].A)
		best := 0
		bestDist := uint32(math.MaxUint32)
		for j := 0; j < 8; j++ {
			d := int16(target) - int16(alpha[j])
			if d < 0 {
				d = -d
			}
			if uint32(d) < bestDist {
				bestDist = uint32(d)
				best = j
			}
		}

		bitOffset := i * 3
		byteIndex := 2 + bitOffset/8
		shift := uint(bitOffset % 8)

		value := uint16(best) << shift
		block[byteIndex] |= byte(value & 0xFF)
		if shift > 5 && byteIndex+1 < 8 {
			block[byteIndex+1] |= byte(value >> 8)
		}
	}
}

func selectEndpoints(colors [16]color.RGBA) (color.RGBA, color.RGBA) {
	var c0, c1 color.RGBA
	maxDist := uint32(0)

	for i := 0; i < len(colors); i++ {
		for j := i + 1; j < len(colors); j++ {
			dist := colorDistance(colors[i], colors[j])
			if dist > maxDist {
				maxDist = dist
				c0 = colors[i]
				c1 = colors[j]
			}
		}
	}

	return c0, c1
}

func colorDistance(c1, c2 color.RGBA) uint32 {
	dr := int32(c1.R) - int32(c2.R)
	dg := int32(c1.G) - int32(c2.G)
	db := int32(c1.B) - int32(c2.B)
	return uint32(dr*dr + dg*dg + db*db)
}

func computeColorIndices(colors [16]color.RGBA, c0, c1 color.RGBA, hasAlpha bool) []byte {
	indices := make([]byte, 4)

	var palette [4]color.RGBA
	palette[0] = c0
	palette[1] = c1

	if hasAlpha {
		palette[2] = interpolateRGB(c0, c1, 2, 1, 3)
		palette[3] = interpolateRGB(c0, c1, 1, 2, 3)
	} else {
		if rgbaToRGB565(c0.R, c0.G, c0.B) > rgbaToRGB565(c1.R, c1.G, c1.B) {
			palette[2] = interpolateRGB(c0, c1, 2, 1, 3)
			palette[3] = interpolateRGB(c0, c1, 1, 2, 3)
		} else {
			palette[2] = interpolateRGB(c0, c1, 1, 1, 2)
			palette[3] = color.RGBA{0, 0, 0, 0}
		}
	}

	for i := 0; i < 16; i++ {
		bestDist := uint32(math.MaxUint32)
		bestIndex := 0
		for j := 0; j < 4; j++ {
			dist := colorDistance(colors[i], palette[j])
			if dist < bestDist {
				bestDist = dist
				bestIndex = j
			}
		}

		bytePos := i / 4
		bitPos := uint((i % 4) * 2)
		indices[bytePos] |= byte(bestIndex) << bitPos
	}

	return indices
}

func generatePalette() [256]color.RGBA {
	var palette [256]color.RGBA
	for i := 0; i < 216; i++ {
		r := uint8((i / 36) * 51)
		g := uint8(((i % 36) / 6) * 51)
		b := uint8((i % 6) * 51)
		palette[i] = color.RGBA{R: r, G: g, B: b, A: 255}
	}
	for i := 216; i < 256; i++ {
		palette[i] = color.RGBA{A: 255}
	}
	return palette
}

func findClosestPaletteIndex(palette [256]color.RGBA, c color.NRGBA) int {
	bestDist := uint32(math.MaxUint32)
	bestIndex := 0

	for i, p := range palette {
		dr := int32(c.R) - int32(p.R)
		dg := int32(c.G) - int32(p.G)
		db := int32(c.B) - int32(p.B)
		dist := uint32(dr*dr + dg*dg + db*db)
		if dist < bestDist {
			bestDist = dist
			bestIndex = i
		}
	}

	return bestIndex
}
