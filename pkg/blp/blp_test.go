package blp

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func makeTestImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x * 37) % 256),
				G: uint8((y * 61) % 256),
				B: uint8((x + y) * 17 % 256),
				A: 255,
			})
		}
	}
	return img
}

func TestBLP2HeaderLayout(t *testing.T) {
	img := makeTestImage(16, 16)

	blp := New()
	if err := blp.EncodeFromImage(img, FormatDXT5); err != nil {
		t.Fatalf("EncodeFromImage: %v", err)
	}
	data, err := blp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if string(data[0:4]) != "BLP2" {
		t.Fatalf("magic = %q, want BLP2", data[0:4])
	}
	if v := binary.LittleEndian.Uint32(data[4:8]); v != 1 {
		t.Fatalf("version = %d, want 1", v)
	}
	if data[8] != colorDXT {
		t.Fatalf("colorEncoding = %d, want %d", data[8], colorDXT)
	}
	if data[9] != 8 {
		t.Fatalf("alphaBitDepth = %d, want 8", data[9])
	}
	if data[10] != alphaDXT5 {
		t.Fatalf("alphaType = %d, want %d", data[10], alphaDXT5)
	}
	if v := binary.LittleEndian.Uint32(data[12:16]); v != 16 {
		t.Fatalf("width = %d, want 16", v)
	}
	if v := binary.LittleEndian.Uint32(data[16:20]); v != 16 {
		t.Fatalf("height = %d, want 16", v)
	}
	if v := binary.LittleEndian.Uint32(data[20:24]); v != blp2TotalHeader {
		t.Fatalf("mipOffsets[0] = %#x, want %#x", v, blp2TotalHeader)
	}
}

func TestBGRARoundTrip(t *testing.T) {
	img := makeTestImage(8, 8)

	blp := New()
	if err := blp.EncodeFromImage(img, FormatARGB8888); err != nil {
		t.Fatalf("EncodeFromImage: %v", err)
	}
	data, err := blp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	if parsed.Format != FormatARGB8888 {
		t.Fatalf("format = %d, want %d", parsed.Format, FormatARGB8888)
	}
	if parsed.Width != 8 || parsed.Height != 8 {
		t.Fatalf("dimensions = %dx%d, want 8x8", parsed.Width, parsed.Height)
	}

	decoded, err := parsed.DecodeMipmap(0)
	if err != nil {
		t.Fatalf("DecodeMipmap: %v", err)
	}

	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			want := img.NRGBAAt(x, y)
			got := decoded.At(x, y).(color.RGBA)
			if got.R != want.R || got.G != want.G || got.B != want.B || got.A != want.A {
				t.Fatalf("pixel (%d,%d) = %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestDXT5DecodeAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < 16; i++ {
		img.SetNRGBA(i%4, i/4, color.NRGBA{R: 128, G: 64, B: 200, A: 0})
	}
	img.SetNRGBA(3, 3, color.NRGBA{R: 128, G: 64, B: 200, A: 255})

	blp := New()
	if err := blp.EncodeFromImage(img, FormatDXT5); err != nil {
		t.Fatalf("EncodeFromImage: %v", err)
	}
	data, err := blp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	decoded, err := parsed.DecodeMipmap(0)
	if err != nil {
		t.Fatalf("DecodeMipmap: %v", err)
	}

	transparent := decoded.At(0, 0).(color.RGBA).A
	opaque := decoded.At(3, 3).(color.RGBA).A
	if transparent > opaque {
		t.Fatalf("alpha ordering wrong: transparent=%d opaque=%d", transparent, opaque)
	}
	if transparent >= 128 {
		t.Fatalf("transparent pixel alpha = %d, want < 128", transparent)
	}
	if opaque <= 128 {
		t.Fatalf("opaque pixel alpha = %d, want > 128", opaque)
	}
}

func TestPalettedRoundTrip(t *testing.T) {
	img := makeTestImage(8, 8)

	blp := New()
	if err := blp.EncodeFromImage(img, FormatPaletted); err != nil {
		t.Fatalf("EncodeFromImage: %v", err)
	}
	data, err := blp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	if parsed.Format != FormatPaletted {
		t.Fatalf("format = %d, want %d", parsed.Format, FormatPaletted)
	}

	decoded, err := parsed.DecodeMipmap(0)
	if err != nil {
		t.Fatalf("DecodeMipmap: %v", err)
	}
	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 8 {
		t.Fatalf("dimensions = %dx%d, want 8x8", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestBLP1RoundTrip(t *testing.T) {
	img := makeTestImage(8, 8)

	blp := New()
	blp.Version = 1
	if err := blp.EncodeFromImage(img, FormatPaletted); err != nil {
		t.Fatalf("EncodeFromImage: %v", err)
	}
	data, err := blp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if !bytes.Equal(data[0:4], BLP1Magic) {
		t.Fatalf("magic = %q, want BLP1", data[0:4])
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	if parsed.Version != 1 || parsed.Format != FormatPaletted {
		t.Fatalf("parsed version/format = %d/%d, want 1/%d", parsed.Version, parsed.Format, FormatPaletted)
	}
	if parsed.Width != 8 || parsed.Height != 8 {
		t.Fatalf("dimensions = %dx%d, want 8x8", parsed.Width, parsed.Height)
	}
}
