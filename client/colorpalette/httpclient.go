package colorpalette

import (
	"github.com/dghubble/sling"

	"context"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"errors"
)

type httpClient struct {
	slingLib *sling.Sling
	baseURL string
	apiKey string
	httpClient *http.Client
}

const (
	GenerateColorPaletteEndpoint = "/default/testColorPalette"
)

func NewHttpClient(slingLib *sling.Sling, baseURL string, apiKey string) *httpClient {
	return &httpClient{
		slingLib: slingLib,
		baseURL: baseURL,
		apiKey: apiKey,
		httpClient: &http.Client{},
	}
}

func (s *httpClient) GenerateColorPalette(ctx context.Context, req ColorPaletteGenerationRequest) error {
	targetURL := fmt.Sprintf("%s%s", s.baseURL, GenerateColorPaletteEndpoint)
	log.Printf("httpClient.GenerateColorPalette target_url=%v", targetURL)
	log.Printf("payload=%v", req.String())
	httpReq, err := sling.New().Set("x-api-key", s.apiKey).Post(targetURL).BodyJSON(req).Request()
	if err != nil {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, err)
	}
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, err)
	}
	if resp == nil {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, err)
	}
	isOk := IsHTTPSuccess(resp.StatusCode)
	if !isOk {
		return fmt.Errorf("httpClient.GenerateColorPalette target=%v err=%v", targetURL, errors.New(string(body)))
	}
	return nil
}

func IsHTTPSuccess(statusCode int) bool {
	return statusCode == http.StatusCreated || statusCode == http.StatusOK
}
