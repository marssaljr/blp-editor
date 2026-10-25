package imageops

import (
	"image"
	"image/color"
	"math"
)

// Operations provides image manipulation functions.
type Operations struct{}

// New creates a new Operations instance.
func New() *Operations {
	return &Operations{}
}

func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

// Resize resizes an image to the specified dimensions using bilinear sampling.
func (o *Operations) Resize(img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if width <= 0 || height <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))

	if width == srcW && height == srcH {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dst.SetNRGBA(x, y, nrgbaAt(img, srcBounds.Min.X+x, srcBounds.Min.Y+y))
			}
		}
		return dst
	}

	xRatio := float64(srcW-1) / float64(max(width-1, 1))
	yRatio := float64(srcH-1) / float64(max(height-1, 1))

	for y := 0; y < height; y++ {
		srcY := float64(y) * yRatio
		y1 := int(srcY)
		y2 := y1 + 1
		if y2 >= srcH {
			y2 = srcH - 1
		}
		dy := srcY - float64(y1)

		for x := 0; x < width; x++ {
			srcX := float64(x) * xRatio
			x1 := int(srcX)
			x2 := x1 + 1
			if x2 >= srcW {
				x2 = srcW - 1
			}
			dx := srcX - float64(x1)

			c11 := nrgbaAt(img, srcBounds.Min.X+x1, srcBounds.Min.Y+y1)
			c21 := nrgbaAt(img, srcBounds.Min.X+x2, srcBounds.Min.Y+y1)
			c12 := nrgbaAt(img, srcBounds.Min.X+x1, srcBounds.Min.Y+y2)
			c22 := nrgbaAt(img, srcBounds.Min.X+x2, srcBounds.Min.Y+y2)

			r := lerp(lerp(float64(c11.R), float64(c21.R), dx), lerp(float64(c12.R), float64(c22.R), dx), dy)
			g := lerp(lerp(float64(c11.G), float64(c21.G), dx), lerp(float64(c12.G), float64(c22.G), dx), dy)
			b := lerp(lerp(float64(c11.B), float64(c21.B), dx), lerp(float64(c12.B), float64(c22.B), dx), dy)
			a := lerp(lerp(float64(c11.A), float64(c21.A), dx), lerp(float64(c12.A), float64(c22.A), dx), dy)

			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clampFloat(r)),
				G: uint8(clampFloat(g)),
				B: uint8(clampFloat(b)),
				A: uint8(clampFloat(a)),
			})
		}
	}

	return dst
}

// Crop crops an image to the specified rectangle.
func (o *Operations) Crop(img image.Image, rect image.Rectangle) image.Image {
	bounds := img.Bounds()
	rect = rect.Intersect(bounds)

	dst := image.NewNRGBA(rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			dst.SetNRGBA(x-rect.Min.X, y-rect.Min.Y, nrgbaAt(img, x, y))
		}
	}
	return dst
}

// Rotate rotates an image by the specified angle in degrees.
func (o *Operations) Rotate(img image.Image, angle float64) image.Image {
	bounds := img.Bounds()
	centerX := float64(bounds.Dx()) / 2
	centerY := float64(bounds.Dy()) / 2

	radians := angle * math.Pi / 180
	cos := math.Cos(radians)
	sin := math.Sin(radians)

	corners := [4]struct{ x, y float64 }{
		{-centerX, -centerY},
		{float64(bounds.Dx()) - centerX, -centerY},
		{-centerX, float64(bounds.Dy()) - centerY},
		{float64(bounds.Dx()) - centerX, float64(bounds.Dy()) - centerY},
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, corner := range corners {
		rotatedX := corner.x*cos - corner.y*sin
		rotatedY := corner.x*sin + corner.y*cos
		minX = math.Min(minX, rotatedX)
		minY = math.Min(minY, rotatedY)
		maxX = math.Max(maxX, rotatedX)
		maxY = math.Max(maxY, rotatedY)
	}

	newWidth := int(maxX - minX + 0.5)
	newHeight := int(maxY - minY + 0.5)

	dst := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	newCenterX := float64(newWidth) / 2
	newCenterY := float64(newHeight) / 2

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := (float64(x)-newCenterX)*cos + (float64(y)-newCenterY)*sin + centerX
			srcY := -(float64(x)-newCenterX)*sin + (float64(y)-newCenterY)*cos + centerY

			if srcX >= 0 && srcX < float64(bounds.Dx()) && srcY >= 0 && srcY < float64(bounds.Dy()) {
				dst.SetNRGBA(x, y, nrgbaAt(img, bounds.Min.X+int(srcX), bounds.Min.Y+int(srcY)))
			}
		}
	}

	return dst
}

// FlipHorizontal flips an image horizontally.
func (o *Operations) FlipHorizontal(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.SetNRGBA(bounds.Max.X-1-x, y, nrgbaAt(img, x, y))
		}
	}
	return dst
}

// FlipVertical flips an image vertically.
func (o *Operations) FlipVertical(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.SetNRGBA(x, bounds.Max.Y-1-y, nrgbaAt(img, x, y))
		}
	}
	return dst
}

// AdjustBrightness adjusts the brightness of an image.
func (o *Operations) AdjustBrightness(img image.Image, factor float64) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			c.R = uint8(clampFloat(float64(c.R) * factor))
			c.G = uint8(clampFloat(float64(c.G) * factor))
			c.B = uint8(clampFloat(float64(c.B) * factor))
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// AdjustContrast adjusts the contrast of an image.
func (o *Operations) AdjustContrast(img image.Image, factor float64) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			c.R = uint8(clampFloat((float64(c.R)-128)*factor + 128))
			c.G = uint8(clampFloat((float64(c.G)-128)*factor + 128))
			c.B = uint8(clampFloat((float64(c.B)-128)*factor + 128))
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// AdjustSaturation adjusts the saturation of an image.
func (o *Operations) AdjustSaturation(img image.Image, factor float64) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			gray := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
			c.R = uint8(clampFloat(gray + factor*(float64(c.R)-gray)))
			c.G = uint8(clampFloat(gray + factor*(float64(c.G)-gray)))
			c.B = uint8(clampFloat(gray + factor*(float64(c.B)-gray)))
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// Invert inverts the colors of an image.
func (o *Operations) Invert(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			c.R = 255 - c.R
			c.G = 255 - c.G
			c.B = 255 - c.B
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// Grayscale converts an image to grayscale.
func (o *Operations) Grayscale(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			gray := uint8(clampFloat(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)))
			c.R, c.G, c.B = gray, gray, gray
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// SetAlpha sets the alpha channel of an image.
func (o *Operations) SetAlpha(img image.Image, alpha uint8) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgbaAt(img, x, y)
			c.A = alpha
			dst.SetNRGBA(x, y, c)
		}
	}
	return dst
}

// ApplyFilter applies a custom filter function to each pixel.
func (o *Operations) ApplyFilter(img image.Image, filter func(color.Color) color.Color) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, filter(img.At(x, y)))
		}
	}
	return dst
}

// Blend blends two images together with the specified weight.
func (o *Operations) Blend(img1, img2 image.Image, weight float64) image.Image {
	bounds := img1.Bounds().Intersect(img2.Bounds())
	dst := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c1 := nrgbaAt(img1, x, y)
			c2 := nrgbaAt(img2, x, y)
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clampFloat(lerp(float64(c1.R), float64(c2.R), weight))),
				G: uint8(clampFloat(lerp(float64(c1.G), float64(c2.G), weight))),
				B: uint8(clampFloat(lerp(float64(c1.B), float64(c2.B), weight))),
				A: uint8(clampFloat(lerp(float64(c1.A), float64(c2.A), weight))),
			})
		}
	}
	return dst
}

// Composite draws overlay on top of base using alpha-over compositing.
// opacity controls the strength of the overlay in the range [0, 1]. Only the
// region where overlay is inside its own bounds is composited.
func (o *Operations) Composite(base, overlay image.Image, opacity float64) image.Image {
	bounds := base.Bounds()
	dst := image.NewNRGBA(bounds)
	overlayBounds := overlay.Bounds()
	opacity = clamp01(opacity)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			b := nrgbaAt(base, x, y)
			if x < overlayBounds.Min.X || x >= overlayBounds.Max.X || y < overlayBounds.Min.Y || y >= overlayBounds.Max.Y {
				dst.SetNRGBA(x, y, b)
				continue
			}

			ov := nrgbaAt(overlay, x, y)
			srcA := (float64(ov.A) / 255) * opacity
			if srcA <= 0 {
				dst.SetNRGBA(x, y, b)
				continue
			}
			if srcA >= 1 {
				dst.SetNRGBA(x, y, ov)
				continue
			}

			dstA := float64(b.A) / 255
			outA := srcA + dstA*(1-srcA)
			if outA <= 0 {
				dst.SetNRGBA(x, y, color.NRGBA{})
				continue
			}

			outR := (float64(ov.R)*srcA + float64(b.R)*dstA*(1-srcA)) / outA
			outG := (float64(ov.G)*srcA + float64(b.G)*dstA*(1-srcA)) / outA
			outB := (float64(ov.B)*srcA + float64(b.B)*dstA*(1-srcA)) / outA

			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(clampFloat(outR)),
				G: uint8(clampFloat(outG)),
				B: uint8(clampFloat(outB)),
				A: uint8(clampFloat(outA * 255)),
			})
		}
	}
	return dst
}

func clampFloat(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}
