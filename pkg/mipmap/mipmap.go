package mipmap

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/marssaljr/go-template-git/pkg/blp"
)

// Generator handles mipmap generation
type Generator struct {
	Filter FilterType
}

// FilterType represents the filtering algorithm for mipmap generation
type FilterType int

const (
	FilterNearest FilterType = iota
	FilterLinear
	FilterCubic
)

// NewGenerator creates a new mipmap generator
func NewGenerator(filter FilterType) *Generator {
	return &Generator{
		Filter: filter,
	}
}

// GenerateMipmaps generates all mipmaps for a BLP
func (g *Generator) GenerateMipmaps(blp *blp.BLP) error {
	if blp.MipmapCount == 0 {
		return errors.New("no base mipmap to generate from")
	}

	// Decode base mipmap
	baseImg, err := blp.DecodeMipmap(0)
	if err != nil {
		return err
	}

	// Calculate required mipmap levels
	maxDim := blp.Width
	if blp.Height > maxDim {
		maxDim = blp.Height
	}
	requiredLevels := uint32(math.Log2(float64(maxDim))) + 1

	// Resize and encode each level
	var allData []byte
	offsets := make([]uint32, 16)
	sizes := make([]uint32, 16)

	currentImg := baseImg
	currentWidth := blp.Width
	currentHeight := blp.Height

	for level := uint32(0); level < requiredLevels; level++ {
		if level > 0 {
			// Resize for next level
			nextWidth := currentWidth / 2
			nextHeight := currentHeight / 2
			if nextWidth < 1 {
				nextWidth = 1
			}
			if nextHeight < 1 {
				nextHeight = 1
			}

			currentImg = g.resize(currentImg, int(nextWidth), int(nextHeight))
			currentWidth = nextWidth
			currentHeight = nextHeight
		}

		// Encode current level
		encoded, err := g.encodeLevel(currentImg, blp.Format)
		if err != nil {
			return err
		}

		offsets[level] = uint32(len(allData))
		sizes[level] = uint32(len(encoded))
		allData = append(allData, encoded...)
	}

	blp.MipmapOffsets = offsets
	blp.MipmapSizes = sizes
	blp.MipmapCount = requiredLevels
	blp.Data = allData

	return nil
}

// resize resizes an image using the selected filter
func (g *Generator) resize(img image.Image, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	switch g.Filter {
	case FilterNearest:
		return resizeNearest(img, width, height)
	case FilterLinear:
		return resizeLinear(img, width, height)
	case FilterCubic:
		return resizeCubic(img, width, height)
	default:
		return resizeLinear(img, width, height)
	}
}

// resizeNearest implements nearest-neighbor resizing
func resizeNearest(img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(srcW) / float64(width)
	yRatio := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)

			if srcX >= srcW {
				srcX = srcW - 1
			}
			if srcY >= srcH {
				srcY = srcH - 1
			}

			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	return dst
}

// resizeLinear implements bilinear resizing
func resizeLinear(img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(srcW-1) / float64(width)
	yRatio := float64(srcH-1) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := float64(x) * xRatio
			srcY := float64(y) * yRatio

			x1 := int(srcX)
			y1 := int(srcY)
			x2 := x1 + 1
			y2 := y1 + 1

			if x2 >= srcW {
				x2 = srcW - 1
			}
			if y2 >= srcH {
				y2 = srcH - 1
			}

			// Get four neighboring pixels
			c11 := colorToNRGBA(img.At(x1, y1))
			c12 := colorToNRGBA(img.At(x1, y2))
			c21 := colorToNRGBA(img.At(x2, y1))
			c22 := colorToNRGBA(img.At(x2, y2))

			// Interpolate
			dx := srcX - float64(x1)
			dy := srcY - float64(y1)

			r := lerp(lerp(float64(c11.R), float64(c21.R), dx), lerp(float64(c12.R), float64(c22.R), dx), dy)
			g := lerp(lerp(float64(c11.G), float64(c21.G), dx), lerp(float64(c12.G), float64(c22.G), dx), dy)
			b := lerp(lerp(float64(c11.B), float64(c21.B), dx), lerp(float64(c12.B), float64(c22.B), dx), dy)
			a := lerp(lerp(float64(c11.A), float64(c21.A), dx), lerp(float64(c12.A), float64(c22.A), dx), dy)

			dst.Set(x, y, color.NRGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: uint8(a),
			})
		}
	}

	return dst
}

// resizeCubic implements bicubic resizing
func resizeCubic(img image.Image, width, height int) image.Image {
	// Simplified bicubic - for production use proper implementation
	return resizeLinear(img, width, height)
}

// lerp performs linear interpolation
func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// colorToNRGBA converts any color to NRGBA
func colorToNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// encodeLevel encodes an image to the specified BLP format
func (g *Generator) encodeLevel(img image.Image, format uint32) ([]byte, error) {
	blp := blp.New()
	blp.Format = format

	bounds := img.Bounds()
	blp.Width = uint32(bounds.Dx())
	blp.Height = uint32(bounds.Dy())

	err := blp.EncodeFromImage(img, format)
	if err != nil {
		return nil, err
	}

	return blp.Data, nil
}

// RemoveMipmaps removes all mipmaps except the base level
func RemoveMipmaps(blp *blp.BLP) error {
	if blp.MipmapCount == 0 {
		return errors.New("no mipmaps to remove")
	}

	// Get base mipmap data
	baseData, err := blp.GetMipmapData(0)
	if err != nil {
		return err
	}

	blp.Data = baseData
	blp.MipmapCount = 1
	blp.MipmapOffsets[0] = 0
	blp.MipmapSizes[0] = uint32(len(baseData))

	// Clear other mipmap info
	for i := 1; i < 16; i++ {
		blp.MipmapOffsets[i] = 0
		blp.MipmapSizes[i] = 0
	}

	return nil
}

// ExtractMipmap extracts a specific mipmap level as a new BLP
func ExtractMipmap(blpFile *blp.BLP, level int) (*blp.BLP, error) {
	if level < 0 || level >= int(blpFile.MipmapCount) {
		return nil, errors.New("invalid mipmap level")
	}

	data, err := blpFile.GetMipmapData(level)
	if err != nil {
		return nil, err
	}

	width, height, err := blpFile.GetMipmapDimensions(level)
	if err != nil {
		return nil, err
	}

	newBlp := blp.New()
	newBlp.Version = blpFile.Version
	newBlp.Format = blpFile.Format
	newBlp.Width = width
	newBlp.Height = height
	newBlp.MipmapCount = 1
	newBlp.MipmapOffsets[0] = 0
	newBlp.MipmapSizes[0] = uint32(len(data))
	newBlp.Data = data

	return newBlp, nil
}

// ResizeMipmap resizes the base mipmap level and discards the remaining
// levels. The caller can regenerate mipmaps afterwards if needed.
func ResizeMipmap(blp *blp.BLP, level int, newWidth, newHeight uint32) error {
	if level != 0 {
		return errors.New("only the base mipmap level can be resized")
	}

	img, err := blp.DecodeMipmap(0)
	if err != nil {
		return err
	}

	generator := NewGenerator(FilterLinear)
	resized := generator.resize(img, int(newWidth), int(newHeight))

	encoded, err := generator.encodeLevel(resized, blp.Format)
	if err != nil {
		return err
	}

	blp.Data = encoded
	blp.MipmapCount = 1
	blp.MipmapOffsets[0] = 0
	blp.MipmapSizes[0] = uint32(len(encoded))
	for i := 1; i < 16; i++ {
		blp.MipmapOffsets[i] = 0
		blp.MipmapSizes[i] = 0
	}
	blp.Width = newWidth
	blp.Height = newHeight
	blp.HasMipmaps = 0

	return nil
}

// SetMipmapData replaces the data for a specific mipmap level
func SetMipmapData(blp *blp.BLP, level int, data []byte) error {
	if level < 0 || level >= 16 {
		return errors.New("invalid mipmap level")
	}

	blp.MipmapOffsets[level] = uint32(len(blp.Data))
	blp.MipmapSizes[level] = uint32(len(data))
	blp.Data = append(blp.Data, data...)

	if level >= int(blp.MipmapCount) {
		blp.MipmapCount = uint32(level + 1)
	}

	return nil
}

// GetMipmapInfo returns information about all mipmaps
func GetMipmapInfo(blp *blp.BLP) []MipmapInfo {
	info := make([]MipmapInfo, blp.MipmapCount)

	for i := uint32(0); i < blp.MipmapCount; i++ {
		width, height, _ := blp.GetMipmapDimensions(int(i))
		size := blp.MipmapSizes[i]

		info[i] = MipmapInfo{
			Level:  int(i),
			Width:  width,
			Height: height,
			Size:   size,
		}
	}

	return info
}

// MipmapInfo contains information about a mipmap level
type MipmapInfo struct {
	Level  int
	Width  uint32
	Height uint32
	Size   uint32
}

// RegenerateMipmap regenerates a specific mipmap level from the base
func RegenerateMipmap(blp *blp.BLP, level int, filter FilterType) error {
	if level < 1 || level >= int(blp.MipmapCount) {
		return errors.New("invalid mipmap level for regeneration")
	}

	// Decode base mipmap
	baseImg, err := blp.DecodeMipmap(0)
	if err != nil {
		return err
	}

	// Generate mipmaps up to target level
	generator := NewGenerator(filter)
	currentImg := baseImg

	for l := 1; l <= level; l++ {
		nextWidth, nextHeight, _ := blp.GetMipmapDimensions(l)
		currentImg = generator.resize(currentImg, int(nextWidth), int(nextHeight))
	}

	// Encode and replace
	encoded, err := generator.encodeLevel(currentImg, blp.Format)
	if err != nil {
		return err
	}

	blp.MipmapOffsets[level] = uint32(len(blp.Data))
	blp.MipmapSizes[level] = uint32(len(encoded))
	blp.Data = append(blp.Data, encoded...)

	return nil
}
