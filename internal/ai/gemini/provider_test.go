package gemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"organizer/internal/ai"
	"organizer/internal/ai/gemini"
)

type TestOutput struct {
	Title string `json:"title"`
	Score int    `json:"score"`
}

func TestGeminiProvider_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "generateContent") {
			t.Errorf("expected path to contain generateContent, got %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "test-gemini-key" {
			t.Errorf("expected x-goog-api-key header test-gemini-key, got %s", r.Header.Get("x-goog-api-key"))
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		genConfig := reqBody["generationConfig"].(map[string]interface{})
		if genConfig["responseMimeType"] != "application/json" {
			t.Errorf("expected responseMimeType application/json, got %v", genConfig["responseMimeType"])
		}

		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{
								"text": `{"title":"gemini-test","score":99}`,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	provider := gemini.NewProvider("test-gemini-key",
		ai.WithBaseURL(ts.URL),
		ai.WithModel("gemini:gemini-1.5-flash"),
		ai.WithTimeout(5*time.Second),
	)

	if provider.Name() != "gemini" {
		t.Errorf("expected name gemini, got %s", provider.Name())
	}

	var out TestOutput
	err := provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
	if err != nil {
		t.Fatalf("GenerateStructured failed: %v", err)
	}

	if out.Title != "gemini-test" || out.Score != 99 {
		t.Errorf("unexpected parsed result: %+v", out)
	}
}

func TestGeminiProvider_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"Invalid request","status":"INVALID_ARGUMENT"}}`))
	}))
	defer ts.Close()

	provider := gemini.NewProvider("bad-key", ai.WithBaseURL(ts.URL))
	var out TestOutput
	err := provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
	if err == nil {
		t.Fatalf("expected error on 400, got nil")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected 400 in error message, got %v", err)
	}
}
