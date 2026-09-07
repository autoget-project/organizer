package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// CallRecord records details of a call to GenerateStructured.
type CallRecord struct {
	Prompt string
	Schema any
}

// Rule defines a mock rule that returns a specific structured object or JSON string when Prompt matches.
type Rule struct {
	// PromptPattern is a regex (IsRegex) or substring to match against the prompt.
	// An empty pattern with IsRegex:false matches every prompt (catch-all).
	PromptPattern string
	IsRegex       bool
	Response      any // Struct or map or JSON string
	Error         error
}

// Provider implements a thread-safe mock ai.Provider for unit and offline integration tests.
type Provider struct {
	mu          sync.RWMutex
	name        string
	calls       []CallRecord
	rules       []Rule
	defaultResp any
	defaultErr  error
}

// NewProvider creates a new Mock Provider.
func NewProvider() *Provider {
	return &Provider{
		name:  "mock",
		calls: make([]CallRecord, 0),
		rules: make([]Rule, 0),
	}
}

func (p *Provider) Name() string {
	if p.name == "" {
		return "mock"
	}
	return p.name
}

// SetName sets the provider name (e.g. to mimic "grok" or "gemini" if needed).
func (p *Provider) SetName(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.name = name
}

// AddRule registers a matching rule for mock responses.
func (p *Provider) AddRule(rule Rule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rules = append(p.rules, rule)
}

// SetDefaultResponse sets the fallback response when no rule matches.
func (p *Provider) SetDefaultResponse(resp any, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultResp = resp
	p.defaultErr = err
}

// Calls returns all recorded calls.
func (p *Provider) Calls() []CallRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	copied := make([]CallRecord, len(p.calls))
	copy(copied, p.calls)
	return copied
}

// Reset clears recorded calls and rules.
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = nil
	p.rules = nil
	p.defaultResp = nil
	p.defaultErr = nil
}

func (p *Provider) GenerateStructuredWithSearch(ctx context.Context, prompt string, schema any, result any) error {
	return p.GenerateStructured(ctx, prompt, schema, result)
}

func (p *Provider) GenerateStructured(ctx context.Context, prompt string, schema any, result any) error {
	p.mu.Lock()
	p.calls = append(p.calls, CallRecord{
		Prompt: prompt,
		Schema: schema,
	})

	var matchedResp any
	var matchedErr error
	found := false

	for _, rule := range p.rules {
		matched := false
		if rule.IsRegex {
			if re, err := regexp.Compile(rule.PromptPattern); err == nil {
				matched = re.MatchString(prompt)
			}
		} else {
			matched = rule.PromptPattern == "" || rule.PromptPattern == prompt || strings.Contains(prompt, rule.PromptPattern)
		}

		if matched {
			matchedResp = rule.Response
			matchedErr = rule.Error
			found = true
			break
		}
	}

	if !found {
		matchedResp = p.defaultResp
		matchedErr = p.defaultErr
	}
	p.mu.Unlock()

	if matchedErr != nil {
		return matchedErr
	}

	if matchedResp == nil {
		return fmt.Errorf("mock provider: no matching rule or default response for prompt: %s", prompt)
	}

	// Unmarshal into result
	switch val := matchedResp.(type) {
	case string:
		return json.Unmarshal([]byte(val), result)
	case []byte:
		return json.Unmarshal(val, result)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("mock provider: failed to marshal response: %w", err)
		}
		return json.Unmarshal(data, result)
	}
}
