package colorpalette

import (
	"context"
)

type Client interface {
	GenerateColorPalette(ctx context.Context, req ColorPaletteGenerationRequest) error
}
