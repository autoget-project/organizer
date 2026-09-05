package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_StreamableHTTP_CallTool(t *testing.T) {
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
			toolRes := ToolCallResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: `{"movie_results": [{"title": "Inception", "release_date": "2010-07-16", "original_language": "en"}]}`,
					},
				},
			}
			resBytes, _ := json.Marshal(toolRes)
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resBytes,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)

		case "search_japanese_porn":
			// Test SSE / streaming format
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
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resBytes,
			}
			data, _ := json.Marshal(resp)
			_, _ = w.Write([]byte("event: message\ndata: " + string(data) + "\n\n"))

		default:
			http.Error(w, "unknown tool", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	ctx := context.Background()

	// 1. Standard JSON-RPC call
	imdbRes, err := client.FindByIMDbID(ctx, "tt1375666")
	if err != nil {
		t.Fatalf("FindByIMDbID failed: %v", err)
	}
	movieResults, ok := imdbRes["movie_results"].([]interface{})
	if !ok || len(movieResults) == 0 {
		t.Fatalf("expected movie_results, got: %v", imdbRes)
	}

	// 2. SSE streaming response
	javRes, err := client.SearchJapanesePorn(ctx, "SSIS-001")
	if err != nil {
		t.Fatalf("SearchJapanesePorn failed: %v", err)
	}
	if javRes["maker"] != "S1" {
		t.Fatalf("expected maker S1, got %v", javRes["maker"])
	}
}
