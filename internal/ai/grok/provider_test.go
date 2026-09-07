package grok_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/ai/grok"
)

type TestOutput struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestGrokProvider_Success(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assertions inside the server goroutine must stay non-fatal.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		var reqBody map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))

		assert.Equal(t, "grok-test", reqBody["model"])
		assert.InDelta(t, 0.1, reqBody["temperature"], 0.0001)

		respFormat, ok := reqBody["response_format"].(map[string]interface{})
		require.True(t, ok, "response_format must be an object")
		assert.Equal(t, "json_schema", respFormat["type"])
		jsonSchema, ok := respFormat["json_schema"].(map[string]interface{})
		require.True(t, ok, "response_format.json_schema must be an object")
		assert.Equal(t, true, jsonSchema["strict"])
		assert.NotEmpty(t, jsonSchema["name"], "response_format.json_schema.name must be set")

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
	t.Cleanup(ts.Close)

	provider := grok.NewProvider("test-api-key",
		ai.WithBaseURL(ts.URL),
		ai.WithModel("xai:grok-test"),
		ai.WithTimeout(5*time.Second),
	)
	assert.Equal(t, "grok", provider.Name())

	var out TestOutput
	require.NoError(t, provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out))
	assert.Equal(t, TestOutput{Name: "test-item", Count: 42}, out)
}

func TestGrokProvider_SearchVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		withSearch bool
	}{
		{"plain generation omits search_parameters", false},
		{"search generation enables search_parameters", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var reqBody map[string]interface{}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))

				if tt.withSearch {
					sp, ok := reqBody["search_parameters"].(map[string]interface{})
					require.True(t, ok, "search_parameters must be an object")
					assert.Equal(t, "on", sp["mode"])
				} else {
					assert.NotContains(t, reqBody, "search_parameters")
				}

				resp := map[string]interface{}{
					"choices": []map[string]interface{}{
						{
							"message": map[string]interface{}{
								"content": `{"name":"search-item","count":7}`,
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			t.Cleanup(ts.Close)

			provider := grok.NewProvider("test-api-key", ai.WithBaseURL(ts.URL))

			var out TestOutput
			var err error
			if tt.withSearch {
				err = provider.GenerateStructuredWithSearch(context.Background(), "test prompt", TestOutput{}, &out)
			} else {
				err = provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
			}
			require.NoError(t, err)
			assert.Equal(t, TestOutput{Name: "search-item", Count: 7}, out)
		})
	}
}

func TestGrokProvider_ImplementsSearchProvider(t *testing.T) {
	t.Parallel()

	provider := grok.NewProvider("test-api-key")
	var _ ai.SearchProvider = provider
}

func TestGrokProvider_ErrorStatus(t *testing.T) {
	t.Parallel()

	statuses := []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError}

	for _, status := range statuses {
		t.Run(fmt.Sprintf("status_%d", status), func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"api_error"}}`))
			}))
			t.Cleanup(ts.Close)

			provider := grok.NewProvider("bad-key", ai.WithBaseURL(ts.URL))

			var out TestOutput
			err := provider.GenerateStructured(context.Background(), "test prompt", TestOutput{}, &out)
			require.Error(t, err)
			assert.Contains(t, err.Error(), strconv.Itoa(status))
		})
	}
}
