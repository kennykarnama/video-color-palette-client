package colorpalette

import (
	"encoding/json"
)

type ColorPaletteGenerationRequest struct {
	SourceURL      string `json:"sourceURL"`
	SourceSerial   string `json:"sourceSerial"`
	PeriodSeconds  float64 `json:"periodSeconds"`
	PaletteSize    int `json:"paletteSize"`
	FunctionType   int `json:"functionType"`
	DestinationURI string `json:"destinationURI"`
}

type ErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func (cp *ColorPaletteGenerationRequest) String() string {
	b , _ := json.Marshal(cp)
	return string(b)
}
