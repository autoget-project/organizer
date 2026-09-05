package grok

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
	DefaultBaseURL = "https://api.x.ai/v1"
	DefaultModel   = "grok-beta"
)

// Provider implements the ai.Provider interface using xAI's Grok (OpenAI compatible).
type Provider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	options    ai.ProviderOptions
}

// NewProvider creates a new Grok Provider.
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
		model = strings.TrimPrefix(options.Model, "xai:")
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
	return "grok"
}

type chatCompletionRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	Temperature    float64             `json:"temperature"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponseFormat struct {
	Type       string                 `json:"type"`
	JSONSchema *chatJSONSchemaWrapper `json:"json_schema,omitempty"`
}

type chatJSONSchemaWrapper struct {
	Name   string        `json:"name"`
	Strict bool          `json:"strict"`
	Schema ai.JSONSchema `json:"schema"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *Provider) GenerateStructured(ctx context.Context, prompt string, schema any, result any) error {
	var jsonSchema ai.JSONSchema
	var err error

	if s, ok := schema.(ai.JSONSchema); ok {
		jsonSchema = s
	} else {
		jsonSchema, err = ai.GenerateStrictJSONSchema(schema)
		if err != nil {
			return fmt.Errorf("failed to generate strict json schema: %w", err)
		}
	}

	reqBody := chatCompletionRequest{
		Model: p.model,
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: p.options.Temperature,
		ResponseFormat: &chatResponseFormat{
			Type: "json_schema",
			JSONSchema: &chatJSONSchemaWrapper{
				Name:   "structured_output",
				Strict: true,
				Schema: jsonSchema,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

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
		return fmt.Errorf("grok API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return fmt.Errorf("failed to unmarshal chat response: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return fmt.Errorf("grok API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Errorf("grok API returned no choices")
	}

	content := chatResp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), result); err != nil {
		return fmt.Errorf("failed to unmarshal structured result: %w (content: %s)", err, content)
	}

	return nil
}
