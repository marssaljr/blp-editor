package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/marssaljr/go-template-git/pkg/blp"
	"github.com/marssaljr/go-template-git/pkg/imageops"
	"github.com/marssaljr/go-template-git/pkg/mipmap"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	currentBLP *blp.BLP
	undoStack  []*blp.BLP
	redoStack  []*blp.BLP
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) pushUndo() {
	if a.currentBLP != nil {
		a.undoStack = append(a.undoStack, a.currentBLP.Clone())
		a.redoStack = nil
	}
}

func (a *App) resetHistory() {
	a.undoStack = nil
	a.redoStack = nil
}

// Undo reverts the most recent edit.
func (a *App) Undo() error {
	if len(a.undoStack) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	if a.currentBLP != nil {
		a.redoStack = append(a.redoStack, a.currentBLP.Clone())
	}
	a.currentBLP = a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]
	return nil
}

// Redo reapplies the most recently undone edit.
func (a *App) Redo() error {
	if len(a.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}
	if a.currentBLP != nil {
		a.undoStack = append(a.undoStack, a.currentBLP.Clone())
	}
	a.currentBLP = a.redoStack[len(a.redoStack)-1]
	a.redoStack = a.redoStack[:len(a.redoStack)-1]
	return nil
}

// CanUndo reports whether an edit is available to undo.
func (a *App) CanUndo() bool {
	return len(a.undoStack) > 0
}

// CanRedo reports whether an edit is available to redo.
func (a *App) CanRedo() bool {
	return len(a.redoStack) > 0
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// FileInfo contains information about a BLP file
type FileInfo struct {
	Version     uint32 `json:"version"`
	Format      uint32 `json:"format"`
	FormatName  string `json:"formatName"`
	Width       uint32 `json:"width"`
	Height      uint32 `json:"height"`
	MipmapCount uint32 `json:"mipmapCount"`
	HasAlpha    bool   `json:"hasAlpha"`
	FileSize    int64  `json:"fileSize"`
}

// OpenBLP opens a BLP file and returns its information
func (a *App) OpenBLP(filePath string) (*FileInfo, error) {
	// Clean the file path
	filePath = filepath.Clean(filePath)

	// If path is relative, make it absolute
	if !filepath.IsAbs(filePath) {
		absPath, err := filepath.Abs(filePath)
		if err == nil {
			filePath = absPath
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	blp, err := blp.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BLP: %w", err)
	}

	a.currentBLP = blp
	a.resetHistory()

	return &FileInfo{
		Version:     blp.Version,
		Format:      blp.Format,
		FormatName:  blp.FormatName(),
		Width:       blp.Width,
		Height:      blp.Height,
		MipmapCount: blp.MipmapCount,
		HasAlpha:    blp.HasAlpha,
		FileSize:    fileInfo.Size(),
	}, nil
}

// GetMipmapPreview returns a base64-encoded PNG preview of a mipmap level
func (a *App) GetMipmapPreview(level int) (string, error) {
	if a.currentBLP == nil {
		return "", fmt.Errorf("no BLP file loaded")
	}

	img, err := a.currentBLP.DecodeMipmap(level)
	if err != nil {
		return "", fmt.Errorf("failed to decode mipmap: %w", err)
	}

	return encodeImageToBase64(img)
}

// GetMipmapInfo returns information about all mipmaps
func (a *App) GetMipmapInfo() ([]mipmap.MipmapInfo, error) {
	if a.currentBLP == nil {
		return nil, fmt.Errorf("no BLP file loaded")
	}

	return mipmap.GetMipmapInfo(a.currentBLP), nil
}

// SaveBLP saves the current BLP to a file
func (a *App) SaveBLP(filePath string) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}

	data, err := a.currentBLP.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode BLP: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ConvertFormat converts the BLP to a different format
func (a *App) ConvertFormat(format uint32) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	hadMipmaps := a.currentBLP.MipmapCount > 1

	// Decode base mipmap
	img, err := a.currentBLP.DecodeMipmap(0)
	if err != nil {
		return fmt.Errorf("failed to decode mipmap: %w", err)
	}

	// Re-encode with new format
	if err := a.currentBLP.EncodeFromImage(img, format); err != nil {
		return fmt.Errorf("failed to encode to new format: %w", err)
	}

	if hadMipmaps {
		generator := mipmap.NewGenerator(mipmap.FilterLinear)
		if err := generator.GenerateMipmaps(a.currentBLP); err != nil {
			return fmt.Errorf("failed to regenerate mipmaps: %w", err)
		}
	}

	return nil
}

// GenerateMipmaps generates all mipmaps for the current BLP
func (a *App) GenerateMipmaps(filterType string) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	var filter mipmap.FilterType
	switch strings.ToLower(filterType) {
	case "nearest":
		filter = mipmap.FilterNearest
	case "linear":
		filter = mipmap.FilterLinear
	case "cubic":
		filter = mipmap.FilterCubic
	default:
		filter = mipmap.FilterLinear
	}

	generator := mipmap.NewGenerator(filter)
	return generator.GenerateMipmaps(a.currentBLP)
}

// RemoveMipmaps removes all mipmaps except the base level
func (a *App) RemoveMipmaps() error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	return mipmap.RemoveMipmaps(a.currentBLP)
}

// Resize resizes the base mipmap
func (a *App) Resize(width, height uint32) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	return mipmap.ResizeMipmap(a.currentBLP, 0, width, height)
}

// ApplyImageOperation applies an image operation
func (a *App) ApplyImageOperation(operation string, params map[string]interface{}) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	hadMipmaps := a.currentBLP.MipmapCount > 1

	// Decode base mipmap
	img, err := a.currentBLP.DecodeMipmap(0)
	if err != nil {
		return fmt.Errorf("failed to decode mipmap: %w", err)
	}

	ops := imageops.New()
	var result image.Image

	switch operation {
	case "brightness":
		factor, ok := params["factor"].(float64)
		if !ok {
			return fmt.Errorf("invalid brightness factor")
		}
		result = ops.AdjustBrightness(img, factor)
	case "contrast":
		factor, ok := params["factor"].(float64)
		if !ok {
			return fmt.Errorf("invalid contrast factor")
		}
		result = ops.AdjustContrast(img, factor)
	case "saturation":
		factor, ok := params["factor"].(float64)
		if !ok {
			return fmt.Errorf("invalid saturation factor")
		}
		result = ops.AdjustSaturation(img, factor)
	case "invert":
		result = ops.Invert(img)
	case "grayscale":
		result = ops.Grayscale(img)
	case "flip-horizontal":
		result = ops.FlipHorizontal(img)
	case "flip-vertical":
		result = ops.FlipVertical(img)
	case "set-alpha":
		alpha, ok := params["alpha"].(float64)
		if !ok {
			return fmt.Errorf("invalid alpha value")
		}
		result = ops.SetAlpha(img, uint8(alpha))
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	// Re-encode
	if err := a.currentBLP.EncodeFromImage(result, a.currentBLP.Format); err != nil {
		return fmt.Errorf("failed to encode after operation: %w", err)
	}

	if hadMipmaps {
		generator := mipmap.NewGenerator(mipmap.FilterLinear)
		if err := generator.GenerateMipmaps(a.currentBLP); err != nil {
			return fmt.Errorf("failed to regenerate mipmaps: %w", err)
		}
	}

	return nil
}

// decodeImageFile decodes a standard image file or a BLP file into an image.
func decodeImageFile(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	if strings.EqualFold(filepath.Ext(filePath), ".blp") {
		parsed, err := blp.Parse(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BLP image: %w", err)
		}
		img, err := parsed.DecodeMipmap(0)
		if err != nil {
			return nil, fmt.Errorf("failed to decode BLP image: %w", err)
		}
		return img, nil
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("unsupported image format (use PNG, JPEG, GIF, or BLP): %w", err)
	}
	return img, nil
}

// ImportImage imports an image file and converts it to BLP
func (a *App) ImportImage(filePath string, format uint32) error {
	img, err := decodeImageFile(filePath)
	if err != nil {
		return err
	}

	a.currentBLP = blp.New()
	a.currentBLP.Format = format

	if err := a.currentBLP.EncodeFromImage(img, format); err != nil {
		return fmt.Errorf("failed to encode image to BLP: %w", err)
	}

	a.resetHistory()
	return nil
}

// ReplaceImage replaces the current texture content with an image file,
// preserving the current format and dimensions.
func (a *App) ReplaceImage(filePath string) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	img, err := decodeImageFile(filePath)
	if err != nil {
		return err
	}

	hadMipmaps := a.currentBLP.MipmapCount > 1
	format := a.currentBLP.Format
	targetW := int(a.currentBLP.Width)
	targetH := int(a.currentBLP.Height)

	ops := imageops.New()
	if img.Bounds().Dx() != targetW || img.Bounds().Dy() != targetH {
		img = ops.Resize(img, targetW, targetH)
	}

	if err := a.currentBLP.EncodeFromImage(img, format); err != nil {
		return fmt.Errorf("failed to encode replacement image: %w", err)
	}

	if hadMipmaps {
		generator := mipmap.NewGenerator(mipmap.FilterLinear)
		if err := generator.GenerateMipmaps(a.currentBLP); err != nil {
			return fmt.Errorf("failed to regenerate mipmaps: %w", err)
		}
	}

	return nil
}

// CompositeImage composites an image on top of the current texture.
func (a *App) CompositeImage(filePath string, opacity float64, fit bool) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}
	a.pushUndo()

	overlay, err := decodeImageFile(filePath)
	if err != nil {
		return err
	}

	base, err := a.currentBLP.DecodeMipmap(0)
	if err != nil {
		return fmt.Errorf("failed to decode current image: %w", err)
	}

	ops := imageops.New()
	if fit {
		overlay = ops.Resize(overlay, int(a.currentBLP.Width), int(a.currentBLP.Height))
	}

	result := ops.Composite(base, overlay, opacity)

	hadMipmaps := a.currentBLP.MipmapCount > 1
	if err := a.currentBLP.EncodeFromImage(result, a.currentBLP.Format); err != nil {
		return fmt.Errorf("failed to encode composite: %w", err)
	}

	if hadMipmaps {
		generator := mipmap.NewGenerator(mipmap.FilterLinear)
		if err := generator.GenerateMipmaps(a.currentBLP); err != nil {
			return fmt.Errorf("failed to regenerate mipmaps: %w", err)
		}
	}

	return nil
}

// ExportPNG exports the current BLP to a PNG file
func (a *App) ExportPNG(filePath string, mipmapLevel int) error {
	if a.currentBLP == nil {
		return fmt.Errorf("no BLP file loaded")
	}

	img, err := a.currentBLP.DecodeMipmap(mipmapLevel)
	if err != nil {
		return fmt.Errorf("failed to decode mipmap: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

// GetSupportedFormats returns a list of supported BLP formats
func (a *App) GetSupportedFormats() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": uint32(blp.FormatPaletted), "name": "Paletted", "description": "8-bit paletted format"},
		{"id": uint32(blp.FormatDXT1), "name": "DXT1", "description": "DXT1/BC1 compressed (no alpha)"},
		{"id": uint32(blp.FormatDXT3), "name": "DXT3", "description": "DXT3/BC2 compressed (explicit alpha)"},
		{"id": uint32(blp.FormatDXT5), "name": "DXT5", "description": "DXT5/BC3 compressed (interpolated alpha)"},
		{"id": uint32(blp.FormatARGB8888), "name": "ARGB8888", "description": "32-bit uncompressed BGRA"},
	}
}

// GetCurrentFileInfo returns information about the currently loaded BLP
func (a *App) GetCurrentFileInfo() (*FileInfo, error) {
	if a.currentBLP == nil {
		return nil, fmt.Errorf("no BLP file loaded")
	}

	return &FileInfo{
		Version:     a.currentBLP.Version,
		Format:      a.currentBLP.Format,
		FormatName:  a.currentBLP.FormatName(),
		Width:       a.currentBLP.Width,
		Height:      a.currentBLP.Height,
		MipmapCount: a.currentBLP.MipmapCount,
		HasAlpha:    a.currentBLP.HasAlpha,
	}, nil
}

// CloseBLP closes the current BLP file
func (a *App) CloseBLP() {
	a.currentBLP = nil
	a.resetHistory()
}

// SelectFile opens a file picker dialog
func (a *App) SelectFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select BLP File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "BLP Files",
				Pattern:     "*.blp",
			},
			{
				DisplayName: "Image Files",
				Pattern:     "*.png;*.jpg;*.jpeg",
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SelectImageFile opens a file picker dialog for image files.
func (a *App) SelectImageFile() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Image",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Image or BLP Files",
				Pattern:     "*.png;*.jpg;*.jpeg;*.blp",
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SelectSaveFile opens a save file dialog
func (a *App) SelectSaveFile(defaultName string) (string, error) {
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "BLP Files",
				Pattern:     "*.blp",
			},
			{
				DisplayName: "PNG Files",
				Pattern:     "*.png",
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// SelectSavePNG opens a save file dialog for PNG
func (a *App) SelectSavePNG(defaultName string) (string, error) {
	selection, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save PNG File",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PNG Files",
				Pattern:     "*.png",
			},
			{
				DisplayName: "All Files",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// encodeImageToBase64 encodes an image to base64 PNG
func encodeImageToBase64(img image.Image) (string, error) {
	var buf strings.Builder
	err := png.Encode(&buf, img)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(buf.String())), nil
}
