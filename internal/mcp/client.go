package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultTimeout is the default request timeout for MCP server calls.
const DefaultTimeout = 30 * time.Second

// maxResponseBytes bounds MCP response body reads (~10MiB) to protect against
// a misbehaving server streaming unbounded data.
const maxResponseBytes = 10 << 20

// JSONRPCRequest represents a JSON-RPC 2.0 request payload.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// CallToolParams represents parameters for tools/call.
type CallToolParams struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("jsonrpc error (code %d): %s", e.Code, e.Message)
}

// ToolCallResult represents the content items returned by tools/call.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is an individual content item (typically text with JSON content).
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Client provides an interface to an MCP server via Streamable HTTP.
// Limitation: no Streamable HTTP session handling (Mcp-Session-Id header /
// initialize handshake); each tools/call is a stateless POST, which is
// sufficient for the stateless metadata server used here.
type Client interface {
	CallTool(ctx context.Context, name string, arguments interface{}) (json.RawMessage, error)
	SearchJapanesePorn(ctx context.Context, javID string) (map[string]interface{}, error)
	FindByIMDbID(ctx context.Context, imdbID string) (map[string]interface{}, error)
	SearchTVShows(ctx context.Context, title string) (map[string]interface{}, error)
	SearchMovies(ctx context.Context, title string) (map[string]interface{}, error)
	WebSearch(ctx context.Context, query string) (map[string]interface{}, error)
}

// HTTPClient implements Client using Streamable HTTP.
type HTTPClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new MCP client.
func NewClient(endpoint string, customHTTP ...*http.Client) *HTTPClient {
	hc := &http.Client{Timeout: DefaultTimeout}
	if len(customHTTP) > 0 && customHTTP[0] != nil {
		hc = customHTTP[0]
	}
	return &HTTPClient{
		endpoint:   endpoint,
		httpClient: hc,
	}
}

// CallTool performs a tools/call invocation over Streamable HTTP (or standard HTTP POST fallback).
func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments interface{}) (json.RawMessage, error) {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      time.Now().UnixNano(),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}

	bodyData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		return nil, fmt.Errorf("mcp server returned status %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read mcp response: %w", err)
	}

	// Detect SSE via the Content-Type header rather than sniffing the body,
	// so a JSON text value containing a "data:" prefix cannot false-positive.
	isSSE := strings.HasPrefix(strings.ToLower(strings.TrimSpace(
		strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0])), "text/event-stream")

	return parseResponseBytes(respBytes, isSSE)
}

func parseResponseBytes(respBytes []byte, isSSE bool) (json.RawMessage, error) {
	// Parse SSE / text/event-stream response
	if isSSE {
		str := string(respBytes)
		for _, line := range strings.Split(str, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataContent != "" && dataContent != "[DONE]" {
					return parseJSONRPCResponse([]byte(dataContent))
				}
			}
		}
	}

	return parseJSONRPCResponse(respBytes)
}

func parseJSONRPCResponse(data []byte) (json.RawMessage, error) {
	// First attempt parsing standard JSON-RPC 2.0 response
	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(data, &rpcResp); err == nil && (rpcResp.Result != nil || rpcResp.Error != nil) {
		if rpcResp.Error != nil {
			return nil, rpcResp.Error
		}

		// Check if result is a ToolCallResult
		var toolRes ToolCallResult
		if err := json.Unmarshal(rpcResp.Result, &toolRes); err == nil && len(toolRes.Content) > 0 {
			if toolRes.IsError {
				return nil, fmt.Errorf("tool returned error: %s", toolRes.Content[0].Text)
			}
			// Attempt to unmarshal text as JSON or return raw text
			textBytes := []byte(toolRes.Content[0].Text)
			if json.Valid(textBytes) {
				return textBytes, nil
			}
			return rpcResp.Result, nil
		}

		return rpcResp.Result, nil
	}

	// Direct tool result or dictionary
	if json.Valid(data) {
		return data, nil
	}

	return nil, fmt.Errorf("unrecognized mcp response format: %s", string(data))
}

func (c *HTTPClient) callToolMap(ctx context.Context, name string, arguments interface{}) (map[string]interface{}, error) {
	raw, err := c.CallTool(ctx, name, arguments)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result into map: %w", err)
	}
	return res, nil
}

func (c *HTTPClient) SearchJapanesePorn(ctx context.Context, javID string) (map[string]interface{}, error) {
	return c.callToolMap(ctx, "search_japanese_porn", map[string]string{"jav_id": javID})
}

func (c *HTTPClient) FindByIMDbID(ctx context.Context, imdbID string) (map[string]interface{}, error) {
	return c.callToolMap(ctx, "find_by_imdb_id", map[string]string{"imdb_id": imdbID})
}

func (c *HTTPClient) SearchTVShows(ctx context.Context, title string) (map[string]interface{}, error) {
	return c.callToolMap(ctx, "search_tv_shows", map[string]string{"title": title})
}

func (c *HTTPClient) SearchMovies(ctx context.Context, title string) (map[string]interface{}, error) {
	return c.callToolMap(ctx, "search_movies", map[string]string{"title": title})
}

func (c *HTTPClient) WebSearch(ctx context.Context, query string) (map[string]interface{}, error) {
	return c.callToolMap(ctx, "web_search", map[string]string{"query": query})
}
