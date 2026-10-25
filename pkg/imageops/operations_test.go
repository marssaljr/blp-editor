package imageops

import (
	"image"
	"image/color"
	"testing"
)

func TestAdjustBrightnessPreservesAlphaColor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 128})

	result := New().AdjustBrightness(img, 1.0).(*image.NRGBA)
	got := result.NRGBAAt(0, 0)
	if got != (color.NRGBA{R: 200, G: 100, B: 50, A: 128}) {
		t.Fatalf("identity brightness changed pixel: %+v", got)
	}
}

func TestCompositeOpaqueOverlay(t *testing.T) {
	base := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	base.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	base.SetNRGBA(1, 0, color.NRGBA{R: 40, G: 50, B: 60, A: 255})

	overlay := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	overlay.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	overlay.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 0})

	result := New().Composite(base, overlay, 1.0).(*image.NRGBA)
	if got := result.NRGBAAt(0, 0); got != (color.NRGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("opaque overlay pixel = %+v", got)
	}
	if got := result.NRGBAAt(1, 0); got != (color.NRGBA{R: 40, G: 50, B: 60, A: 255}) {
		t.Fatalf("transparent overlay pixel = %+v", got)
	}
}

func TestCompositeHalfOpacity(t *testing.T) {
	base := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	base.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})

	overlay := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	overlay.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	result := New().Composite(base, overlay, 0.5).(*image.NRGBA)
	got := result.NRGBAAt(0, 0)
	if got.R != 127 || got.G != 127 || got.B != 127 {
		t.Fatalf("half opacity blend = %+v, want 127/127/127", got)
	}
}
