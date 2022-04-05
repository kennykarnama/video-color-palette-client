package colorpalette

import (
	"encoding/json"
)

type ColorPaletteGenerationRequest struct {
	SourceURL      string `json:"sourceURL" csv:"source_url"`
	SourceSerial   string `json:"sourceSerial" csv:"source_serial"`
	PeriodSeconds  float64 `json:"periodSeconds" csv:"period_seconds"`
	PaletteSize    int `json:"paletteSize" csv:"palette_size"`
	FunctionType   int `json:"functionType" csv:"function_type"`
	DestinationURI string `json:"destinationURI" csv:"destination_uri"`
}

type ErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func (cp *ColorPaletteGenerationRequest) String() string {
	b , _ := json.Marshal(cp)
	return string(b)
}
