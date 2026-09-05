package grok_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"organizer/internal/ai"
	"organizer/internal/ai/grok"
)

type TestOutput struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestGrokProvider_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected Bearer test-api-key, got %s", auth)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Verify model and temperature
		if reqBody["temperature"].(float64) != 0.1 {
			t.Errorf("expected temperature 0.1, got %v", reqBody["temperature"])
		}

		respFormat := reqBody["response_format"].(map[string]interface{})
		if respFormat["type"] != "json_schema" {
			t.Errorf("expected response_format.type json_schema, got %v", respFormat["type"])
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": `{"name":"test-item","count":42}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	provider := grok.NewProvider("test-api-key",
		ai.WithBaseURL(ts.URL),
		ai.WithModel("xai:grok-test"),
		ai.WithTimeout(5*time.Second),
	)

	if provider.Name() != "grok" {
		t.Errorf("expected name grok, got %s", provider.Name())
	}

	var out TestOutput
	err := provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
	if err != nil {
		t.Fatalf("GenerateStructured failed: %v", err)
	}

	if out.Name != "test-item" || out.Count != 42 {
		t.Errorf("unexpected parsed result: %+v", out)
	}
}

func TestGrokProvider_ErrorStatus(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"api_error"}}`))
			}))
			defer ts.Close()

			provider := grok.NewProvider("bad-key", ai.WithBaseURL(ts.URL))
			var out TestOutput
			err := provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
			if err == nil {
				t.Fatalf("expected error on %d, got nil", status)
			}
			if !strings.Contains(err.Error(), strconv.Itoa(status)) {
				t.Errorf("expected status %d in error message, got %v", status, err)
			}
		})
	}
}
