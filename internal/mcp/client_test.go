package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_StreamableHTTP_CallTool(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		paramsBytes, _ := json.Marshal(req.Params)
		var toolParams CallToolParams
		_ = json.Unmarshal(paramsBytes, &toolParams)

		switch toolParams.Name {
		case "find_by_imdb_id":
			// Standard JSON-RPC response.
			toolRes := ToolCallResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: `{"movie_results": [{"title": "Inception", "release_date": "2010-07-16", "original_language": "en"}]}`,
					},
				},
			}
			resBytes, _ := json.Marshal(toolRes)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resBytes})

		case "search_japanese_porn":
			// SSE / streaming response format.
			w.Header().Set("Content-Type", "text/event-stream")
			toolRes := ToolCallResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: `{"actresses": ["Yua Mikami"], "maker": "S1", "is_vr": false}`,
					},
				},
			}
			resBytes, _ := json.Marshal(toolRes)
			data, _ := json.Marshal(JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resBytes})
			_, _ = w.Write([]byte("event: message\ndata: " + string(data) + "\n\n"))

		default:
			http.Error(w, "unknown tool", http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	client := NewClient(ts.URL)
	ctx := context.Background()

	imdbRes, err := client.FindByIMDbID(ctx, "tt1375666")
	require.NoError(t, err)
	movieResults, ok := imdbRes["movie_results"].([]interface{})
	require.True(t, ok, "movie_results must be a list, got: %v", imdbRes)
	assert.NotEmpty(t, movieResults)

	javRes, err := client.SearchJapanesePorn(ctx, "SSIS-001")
	require.NoError(t, err)
	assert.Equal(t, "S1", javRes["maker"])
}
