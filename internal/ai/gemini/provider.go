// Package gemini implements the ai.Provider interface on top of the official
// Google GenAI SDK (Gemini API backend).
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/genai"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/ptr"
)

const DefaultModel = "gemini-1.5-pro"

// Provider implements the ai.Provider interface using Google Gemini.
type Provider struct {
	client  *genai.Client
	model   string
	options ai.ProviderOptions
}

// NewProvider creates a new Gemini Provider.
func NewProvider(apiKey string, opts ...ai.Option) (*Provider, error) {
	options := ai.DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	model := DefaultModel
	if options.Model != "" {
		model = strings.TrimPrefix(options.Model, "gemini:")
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: &http.Client{Timeout: options.Timeout},
		HTTPOptions: genai.HTTPOptions{
			BaseURL: strings.TrimRight(options.BaseURL, "/"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Provider{
		client:  client,
		model:   model,
		options: options,
	}, nil
}

func (p *Provider) Name() string {
	return "gemini"
}

func (p *Provider) GenerateStructured(ctx context.Context, prompt string, schema any, result any) error {
	return p.generateWithOptionalSearch(ctx, prompt, schema, result, false)
}

func (p *Provider) GenerateStructuredWithSearch(ctx context.Context, prompt string, schema any, result any) error {
	return p.generateWithOptionalSearch(ctx, prompt, schema, result, true)
}

func (p *Provider) generateWithOptionalSearch(ctx context.Context, prompt string, schema any, result any, enableSearch bool) error {
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

	responseSchema, err := toGenaiSchema(jsonSchema)
	if err != nil {
		return fmt.Errorf("failed to convert response schema: %w", err)
	}

	cfg := &genai.GenerateContentConfig{
		Temperature:      ptr.Float32(float32(p.options.Temperature)),
		ResponseMIMEType: "application/json",
		ResponseSchema:   responseSchema,
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}
	if enableSearch {
		cfg.Tools = []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		}
	}

	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(prompt), cfg)
	if err != nil {
		return fmt.Errorf("gemini API request failed: %w", err)
	}

	content := resp.Text()
	if content == "" {
		fullResp, _ := json.Marshal(resp)
		return fmt.Errorf("gemini API returned no text parts in candidate: resp=%s", string(fullResp))
	}

	if err := json.Unmarshal([]byte(content), result); err != nil {
		return fmt.Errorf("failed to unmarshal structured result: %w (content: %s)", err, content)
	}

	return nil
}

// toGenaiSchema converts the project's JSON Schema representation into the
// SDK's Schema type via a JSON round-trip (identical wire field names).
func toGenaiSchema(s ai.JSONSchema) (*genai.Schema, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var out genai.Schema
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
