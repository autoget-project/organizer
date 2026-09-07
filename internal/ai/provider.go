package ai

import (
	"context"
	"time"
)

// DefaultTemperature is the default sampling temperature for structured outputs (0.1 for high determinism).
const DefaultTemperature = 0.1

// DefaultTimeout is the default timeout for AI provider requests.
const DefaultTimeout = 60 * time.Second

// Provider is the unified abstraction for AI structured generation backends.
type Provider interface {
	Name() string
	GenerateStructured(ctx context.Context, prompt string, schema any, result any) error
}

// SearchProvider is an optional interface implemented by providers that support built-in web search (grounding).
type SearchProvider interface {
	Provider
	GenerateStructuredWithSearch(ctx context.Context, prompt string, schema any, result any) error
}

// Option represents a functional option for configuring a Provider.
type Option func(*ProviderOptions)

// ProviderOptions holds configuration options for an AI provider call or client.
type ProviderOptions struct {
	BaseURL     string
	Temperature float64
	Timeout     time.Duration
	Model       string
}

// WithBaseURL overrides the default provider API base URL (essential for httptest mocks).
func WithBaseURL(baseURL string) Option {
	return func(o *ProviderOptions) {
		o.BaseURL = baseURL
	}
}

// WithTemperature sets the model sampling temperature.
func WithTemperature(temp float64) Option {
	return func(o *ProviderOptions) {
		o.Temperature = temp
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *ProviderOptions) {
		o.Timeout = timeout
	}
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(o *ProviderOptions) {
		o.Model = model
	}
}

// DefaultOptions returns the default options for a provider call.
func DefaultOptions() ProviderOptions {
	return ProviderOptions{
		Temperature: DefaultTemperature,
		Timeout:     DefaultTimeout,
	}
}
