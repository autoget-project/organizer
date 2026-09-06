package gemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/ai/gemini"
)

type TestOutput struct {
	Title string `json:"title"`
	Score int    `json:"score"`
}

func TestGeminiProvider_Success(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assertions inside the server goroutine must stay non-fatal.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "generateContent")
		assert.Equal(t, "test-gemini-key", r.Header.Get("x-goog-api-key"))

		var reqBody map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))

		genConfig, ok := reqBody["generationConfig"].(map[string]interface{})
		require.True(t, ok, "generationConfig must be an object")
		assert.Equal(t, "application/json", genConfig["responseMimeType"])

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
	t.Cleanup(ts.Close)

	provider, err := gemini.NewProvider("test-gemini-key",
		ai.WithBaseURL(ts.URL),
		ai.WithModel("gemini:gemini-1.5-flash"),
		ai.WithTimeout(5*time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, "gemini", provider.Name())

	var out TestOutput
	require.NoError(t, provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out))
	assert.Equal(t, TestOutput{Title: "gemini-test", Score: 99}, out)
}

func TestGeminiProvider_APIError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"Invalid request","status":"INVALID_ARGUMENT"}}`))
	}))
	t.Cleanup(ts.Close)

	provider, err := gemini.NewProvider("bad-key", ai.WithBaseURL(ts.URL))
	require.NoError(t, err)

	var out TestOutput
	err = provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "400"), "error must mention the status code, got %v", err)
}
