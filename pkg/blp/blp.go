package blp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

// BLP represents a BLP texture file.
type BLP struct {
	Version       uint32
	Format        uint32
	Width         uint32
	Height        uint32
	MipmapCount   uint32
	MipmapSizes   []uint32
	MipmapOffsets []uint32
	Palette       [256]color.RGBA
	HasAlpha      bool
	Data          []byte

	// On-disk encoding details. These are populated by the parser and used by
	// the encoder so that files round-trip without losing format information.
	ColorEncoding uint8
	AlphaBitDepth uint8
	AlphaType     uint8
	HasMipmaps    uint8
}

// Logical format constants used by the UI and public API. They are mapped to
// BLP2's colorEncoding/alphaType fields when reading and writing files.
const (
	FormatPaletted    = 0
	FormatDXT1        = 1
	FormatDXT3        = 2
	FormatDXT5        = 3
	FormatARGB8888    = 4
	FormatARGB1555    = 5
	FormatARGB4444    = 6
	FormatRGB565      = 7
	FormatARGB2565    = 8
	FormatARGB2101010 = 9
	FormatBC1         = 10
	FormatBC2         = 11
	FormatBC3         = 12
	FormatBC4         = 13
	FormatBC5         = 14
	FormatBC6H        = 15
	FormatBC7         = 16
)

// On-disk BLP2 color encoding values.
const (
	colorJPEG     uint8 = 0
	colorPaletted uint8 = 1
	colorDXT      uint8 = 2
	colorBGRA     uint8 = 3
	colorBGRADup  uint8 = 4
)

// On-disk BLP2 DXT alpha type values.
const (
	alphaDXT1 uint8 = 0
	alphaDXT3 uint8 = 1
	alphaDXT5 uint8 = 7
)

// BLP1 magic number.
var BLP1Magic = []byte("BLP1")

// BLP2 magic number.
var BLP2Magic = []byte("BLP2")

// BLP2 header layout constants.
const (
	blp2HeaderSize  = 0x94  // 148 bytes without the always-present palette
	blp2PaletteSize = 0x400 // 256 * 4
	blp2TotalHeader = 0x494 // 1172 bytes
	blp1HeaderSize  = 0x9C  // 156 bytes without palette
	blp1PaletteSize = 0x400
	blp1TotalHeader = 0x49C
)

// Parse reads a BLP file from a reader.
func Parse(r io.Reader) (*BLP, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read BLP data: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses BLP data from a byte slice.
func ParseBytes(data []byte) (*BLP, error) {
	if len(data) < 4 {
		return nil, errors.New("invalid BLP: too short")
	}

	magic := data[0:4]
	if bytes.Equal(magic, BLP1Magic) {
		return parseBLP1(data)
	}
	if bytes.Equal(magic, BLP2Magic) {
		return parseBLP2(data)
	}

	return nil, errors.New("invalid BLP: unknown magic number")
}

// parseBLP1 parses the BLP1 format used by Warcraft III.
func parseBLP1(data []byte) (*BLP, error) {
	if len(data) < blp1HeaderSize {
		return nil, errors.New("invalid BLP1: header too short")
	}

	content := binary.LittleEndian.Uint32(data[4:8])
	if content != 1 {
		return nil, fmt.Errorf("unsupported BLP1 content type %d (only palettized is supported)", content)
	}

	blp := &BLP{
		Version:       1,
		Format:        FormatPaletted,
		ColorEncoding: colorPaletted,
		AlphaBitDepth: uint8(binary.LittleEndian.Uint32(data[8:12])),
		Width:         binary.LittleEndian.Uint32(data[12:16]),
		Height:        binary.LittleEndian.Uint32(data[16:20]),
		HasMipmaps:    0,
		MipmapOffsets: make([]uint32, 16),
		MipmapSizes:   make([]uint32, 16),
	}

	if binary.LittleEndian.Uint32(data[24:28]) != 0 {
		blp.HasMipmaps = 1
	}

	for i := 0; i < 16; i++ {
		blp.MipmapOffsets[i] = binary.LittleEndian.Uint32(data[28+i*4 : 32+i*4])
		blp.MipmapSizes[i] = binary.LittleEndian.Uint32(data[92+i*4 : 96+i*4])
	}

	for i := 0; i < 256; i++ {
		off := blp1HeaderSize + i*4
		if off+4 > len(data) {
			break
		}
		blp.Palette[i] = color.RGBA{R: data[off+2], G: data[off+1], B: data[off], A: 255}
	}

	blp.HasAlpha = blp.AlphaBitDepth > 0
	if err := blp.normalizePayload(data); err != nil {
		return nil, err
	}
	blp.MipmapCount = determineMipmapCount(blp)
	return blp, nil
}

// parseBLP2 parses the BLP2 format used by World of Warcraft.
func parseBLP2(data []byte) (*BLP, error) {
	if len(data) < blp2HeaderSize {
		return nil, errors.New("invalid BLP2: header too short")
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 1 {
		return nil, fmt.Errorf("unsupported BLP2 version %d", version)
	}

	blp := &BLP{
		Version:       2,
		ColorEncoding: data[8],
		AlphaBitDepth: data[9],
		AlphaType:     data[10],
		HasMipmaps:    data[11],
		Width:         binary.LittleEndian.Uint32(data[12:16]),
		Height:        binary.LittleEndian.Uint32(data[16:20]),
		MipmapOffsets: make([]uint32, 16),
		MipmapSizes:   make([]uint32, 16),
	}

	if blp.ColorEncoding == colorJPEG {
		return nil, errors.New("unsupported BLP2: JPEG-compressed content")
	}

	for i := 0; i < 16; i++ {
		blp.MipmapOffsets[i] = binary.LittleEndian.Uint32(data[20+i*4 : 24+i*4])
		blp.MipmapSizes[i] = binary.LittleEndian.Uint32(data[84+i*4 : 88+i*4])
	}

	for i := 0; i < 256; i++ {
		off := blp2HeaderSize + i*4
		if off+4 > len(data) {
			break
		}
		blp.Palette[i] = color.RGBA{R: data[off+2], G: data[off+1], B: data[off], A: 255}
	}

	blp.Format = formatFromDisk(blp.ColorEncoding, blp.AlphaType)
	blp.HasAlpha = blp.AlphaBitDepth > 0 || blp.AlphaType == alphaDXT3 || blp.AlphaType == alphaDXT5

	if err := blp.normalizePayload(data); err != nil {
		return nil, err
	}
	blp.MipmapCount = determineMipmapCount(blp)
	return blp, nil
}

// normalizePayload rebuilds Data as a compact, sequentially ordered payload and
// converts file-relative mipmap offsets into payload-relative offsets.
func (blp *BLP) normalizePayload(fileData []byte) error {
	payload := make([]byte, 0, len(fileData))
	offsets := make([]uint32, 16)
	sizes := make([]uint32, 16)

	for i := 0; i < 16; i++ {
		off := blp.MipmapOffsets[i]
		size := blp.MipmapSizes[i]
		if size == 0 {
			continue
		}
		if off > uint32(len(fileData)) {
			break
		}
		end := off + size
		if end > uint32(len(fileData)) {
			end = uint32(len(fileData))
		}
		offsets[i] = uint32(len(payload))
		sizes[i] = end - off
		payload = append(payload, fileData[off:end]...)
	}

	blp.MipmapOffsets = offsets
	blp.MipmapSizes = sizes
	blp.Data = payload
	return nil
}

func determineMipmapCount(blp *BLP) uint32 {
	count := uint32(0)
	for i := 0; i < 16; i++ {
		if blp.MipmapSizes[i] == 0 {
			break
		}
		count = uint32(i + 1)
	}
	if count == 0 {
		count = 1
	}
	return count
}

// Encode encodes a BLP structure to bytes.
func (blp *BLP) Encode() ([]byte, error) {
	if blp.Version == 1 {
		return blp.encodeBLP1()
	}
	return blp.encodeBLP2()
}

func (blp *BLP) diskFields() (colorEncoding, alphaBitDepth, alphaType uint8) {
	if blp.ColorEncoding == 0 {
		return diskFieldsFromFormat(blp.Format)
	}
	return blp.ColorEncoding, blp.AlphaBitDepth, blp.AlphaType
}

// encodeBLP2 encodes the BLP2 format.
func (blp *BLP) encodeBLP2() ([]byte, error) {
	colorEncoding, alphaBitDepth, alphaType := blp.diskFields()

	body, fileOffsets, fileSizes := blp.buildPayload(blp2TotalHeader)

	buf := new(bytes.Buffer)
	buf.Write(BLP2Magic)
	binary.Write(buf, binary.LittleEndian, uint32(1)) // version
	buf.WriteByte(colorEncoding)
	buf.WriteByte(alphaBitDepth)
	buf.WriteByte(alphaType)
	buf.WriteByte(blp.mipmapFlag())
	binary.Write(buf, binary.LittleEndian, blp.Width)
	binary.Write(buf, binary.LittleEndian, blp.Height)
	for i := 0; i < 16; i++ {
		binary.Write(buf, binary.LittleEndian, fileOffsets[i])
	}
	for i := 0; i < 16; i++ {
		binary.Write(buf, binary.LittleEndian, fileSizes[i])
	}
	writePalette(buf, blp.Palette)
	buf.Write(body)
	return buf.Bytes(), nil
}

// encodeBLP1 encodes the BLP1 format (palettized content only).
func (blp *BLP) encodeBLP1() ([]byte, error) {
	body, fileOffsets, fileSizes := blp.buildPayload(blp1TotalHeader)

	buf := new(bytes.Buffer)
	buf.Write(BLP1Magic)
	binary.Write(buf, binary.LittleEndian, uint32(1)) // content: direct/palettized
	binary.Write(buf, binary.LittleEndian, uint32(blp.AlphaBitDepth))
	binary.Write(buf, binary.LittleEndian, blp.Width)
	binary.Write(buf, binary.LittleEndian, blp.Height)
	binary.Write(buf, binary.LittleEndian, uint32(5)) // extra
	if blp.MipmapCount > 1 {
		binary.Write(buf, binary.LittleEndian, uint32(1))
	} else {
		binary.Write(buf, binary.LittleEndian, uint32(0))
	}
	for i := 0; i < 16; i++ {
		binary.Write(buf, binary.LittleEndian, fileOffsets[i])
	}
	for i := 0; i < 16; i++ {
		binary.Write(buf, binary.LittleEndian, fileSizes[i])
	}
	writePalette(buf, blp.Palette)
	buf.Write(body)
	return buf.Bytes(), nil
}

// buildPayload concatenates all mipmap levels and computes file-relative
// offsets for the header.
func (blp *BLP) buildPayload(headerSize int) (body []byte, fileOffsets, fileSizes [16]uint32) {
	var buf bytes.Buffer
	for i := 0; i < 16; i++ {
		if uint32(i) >= blp.MipmapCount {
			break
		}
		data, err := blp.GetMipmapData(i)
		if err != nil {
			break
		}
		fileOffsets[i] = uint32(headerSize + buf.Len())
		fileSizes[i] = uint32(len(data))
		buf.Write(data)
	}
	return buf.Bytes(), fileOffsets, fileSizes
}

func (blp *BLP) mipmapFlag() uint8 {
	if blp.MipmapCount > 1 {
		return 1
	}
	return 0
}

func writePalette(buf *bytes.Buffer, palette [256]color.RGBA) {
	for _, c := range palette {
		buf.WriteByte(c.B)
		buf.WriteByte(c.G)
		buf.WriteByte(c.R)
		buf.WriteByte(0)
	}
}

// diskFieldsFromFormat returns the BLP2 on-disk fields for a logical format.
func diskFieldsFromFormat(format uint32) (colorEncoding, alphaBitDepth, alphaType uint8) {
	switch format {
	case FormatPaletted:
		return colorPaletted, 0, 2
	case FormatDXT1:
		return colorDXT, 0, alphaDXT1
	case FormatDXT3:
		return colorDXT, 4, alphaDXT3
	case FormatDXT5:
		return colorDXT, 8, alphaDXT5
	case FormatARGB8888:
		return colorBGRA, 8, 0
	default:
		return colorDXT, 8, alphaDXT5
	}
}

// formatFromDisk returns the logical format for BLP2 on-disk fields.
func formatFromDisk(colorEncoding, alphaType uint8) uint32 {
	switch colorEncoding {
	case colorPaletted:
		return FormatPaletted
	case colorBGRA, colorBGRADup:
		return FormatARGB8888
	case colorDXT:
		switch alphaType {
		case alphaDXT3:
			return FormatDXT3
		case alphaDXT5:
			return FormatDXT5
		default:
			return FormatDXT1
		}
	default:
		return FormatARGB8888
	}
}

// GetMipmapData returns the payload data for a specific mipmap level.
func (blp *BLP) GetMipmapData(level int) ([]byte, error) {
	if level < 0 || level >= 16 || level >= len(blp.MipmapOffsets) || level >= len(blp.MipmapSizes) {
		return nil, fmt.Errorf("invalid mipmap level: %d", level)
	}
	if uint32(level) >= blp.MipmapCount {
		return nil, fmt.Errorf("mipmap level out of range: %d", level)
	}

	offset := blp.MipmapOffsets[level]
	size := blp.MipmapSizes[level]
	if size == 0 {
		return nil, fmt.Errorf("empty mipmap level: %d", level)
	}
	if offset > uint32(len(blp.Data)) {
		return nil, errors.New("mipmap data out of bounds")
	}
	end := offset + size
	if end > uint32(len(blp.Data)) {
		end = uint32(len(blp.Data))
	}
	return blp.Data[offset:end], nil
}

// GetMipmapDimensions returns the dimensions of a specific mipmap level.
func (blp *BLP) GetMipmapDimensions(level int) (uint32, uint32, error) {
	if level < 0 || level >= int(blp.MipmapCount) {
		return 0, 0, fmt.Errorf("invalid mipmap level: %d", level)
	}

	width := blp.Width >> uint(level)
	height := blp.Height >> uint(level)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return width, height, nil
}

// FormatName returns the human-readable name of the logical format.
func (blp *BLP) FormatName() string {
	return FormatName(blp.Format)
}

// FormatName returns the human-readable name for a format value.
func FormatName(format uint32) string {
	switch format {
	case FormatPaletted:
		return "Paletted"
	case FormatDXT1:
		return "DXT1"
	case FormatDXT3:
		return "DXT3"
	case FormatDXT5:
		return "DXT5"
	case FormatARGB8888:
		return "ARGB8888"
	case FormatARGB1555:
		return "ARGB1555"
	case FormatARGB4444:
		return "ARGB4444"
	case FormatRGB565:
		return "RGB565"
	case FormatARGB2565:
		return "ARGB2565"
	case FormatARGB2101010:
		return "ARGB2101010"
	case FormatBC1:
		return "BC1"
	case FormatBC2:
		return "BC2"
	case FormatBC3:
		return "BC3"
	case FormatBC4:
		return "BC4"
	case FormatBC5:
		return "BC5"
	case FormatBC6H:
		return "BC6H"
	case FormatBC7:
		return "BC7"
	default:
		return fmt.Sprintf("Unknown(%d)", format)
	}
}

// ToImage converts the base mipmap of the BLP to a Go image.Image.
func (blp *BLP) ToImage() (image.Image, error) {
	return blp.DecodeMipmap(0)
}

// New creates a new empty BLP structure with sensible defaults.
func New() *BLP {
	return &BLP{
		Version:       2,
		Format:        FormatDXT5,
		MipmapOffsets: make([]uint32, 16),
		MipmapSizes:   make([]uint32, 16),
	}
}

// Clone returns a deep copy of the BLP.
func (blp *BLP) Clone() *BLP {
	clone := *blp
	clone.MipmapOffsets = append([]uint32(nil), blp.MipmapOffsets...)
	clone.MipmapSizes = append([]uint32(nil), blp.MipmapSizes...)
	clone.Data = append([]byte(nil), blp.Data...)
	return &clone
}
