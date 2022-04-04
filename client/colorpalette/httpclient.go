package colorpalette

import (
	"github.com/dghubble/sling"

	"context"
	"fmt"
	"log"
)

type httpClient struct {
	slingLib *sling.Sling
	baseURL string
}

const (
	GenerateColorPaletteEndpoint = ""
)

func NewHttpClient(slingLib *sling.Sling, baseURL string) *httpClient {
	return &httpClient{
		slingLib: slingLib,
		baseURL: baseURL,
	}
}

func (s *httpClient) GenerateColorPalette(ctx context.Context, req ColorPaletteGenerationRequest) error {
	targetURL := fmt.Sprintf("%s%s", s.baseURL, GenerateColorPaletteEndpoint)
	log.Printf("httpClient.GenerateColorPalette target_url=%v", targetURL)
	_, err := s.slingLib.Post(targetURL).BodyJSON(req).Request()
	if err != nil {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, err)
	}
	return nil
}
