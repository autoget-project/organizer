package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"organizer/internal/ai"
)

const (
	DefaultBaseURL = "https://generativelanguage.googleapis.com"
	DefaultModel   = "gemini-1.5-pro"
)

// Provider implements the ai.Provider interface using Google Gemini REST API.
type Provider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	options    ai.ProviderOptions
}

// NewProvider creates a new Gemini Provider.
func NewProvider(apiKey string, opts ...ai.Option) *Provider {
	options := ai.DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	baseURL := DefaultBaseURL
	if options.BaseURL != "" {
		baseURL = strings.TrimRight(options.BaseURL, "/")
	}

	model := DefaultModel
	if options.Model != "" {
		model = strings.TrimPrefix(options.Model, "gemini:")
	}

	return &Provider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: options.Timeout},
		options:    options,
	}
}

func (p *Provider) Name() string {
	return "gemini"
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature      float64        `json:"temperature"`
	ResponseMimeType string         `json:"responseMimeType,omitempty"`
	ResponseSchema   *ai.JSONSchema `json:"responseSchema,omitempty"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (p *Provider) GenerateStructured(ctx context.Context, prompt string, schema any, result any) error {
	var jsonSchema ai.JSONSchema
	var err error

	if s, ok := schema.(ai.JSONSchema); ok {
		jsonSchema = s
	} else {
		jsonSchema, err = ai.GenerateOpenAPISchema(schema)
		if err != nil {
			return fmt.Errorf("failed to generate openapi schema: %w", err)
		}
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      p.options.Temperature,
			ResponseMimeType: "application/json",
			ResponseSchema:   &jsonSchema,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	if geminiResp.Error != nil && geminiResp.Error.Message != "" {
		return fmt.Errorf("gemini API error (%d): %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("gemini API returned no text parts in candidate")
	}

	content := geminiResp.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(content), result); err != nil {
		return fmt.Errorf("failed to unmarshal structured result: %w (content: %s)", err, content)
	}

	return nil
}
