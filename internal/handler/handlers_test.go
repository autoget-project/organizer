package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"organizer/internal/ai"
	"organizer/internal/ai/mock"
	"organizer/internal/model"
	"organizer/internal/pipeline"
	"organizer/internal/pipeline/stage2_enricher"
	"organizer/internal/service"
)

// env is an offline test server wiring every REST endpoint around a mock
// provider and isolated temporary directories.
type env struct {
	mux         http.Handler
	downloadDir string
	targetDir   string
	provider    *mock.Provider
}

func newTestEnv(t *testing.T, prov *mock.Provider) *env {
	t.Helper()
	downloadDir := t.TempDir()
	targetDir := t.TempDir()

	pipe := pipeline.NewPipeline(prov, stage2enricher.NewEnricher(nil, nil, nil), downloadDir)
	exec := service.NewExecutor(downloadDir, targetDir)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/plan", NewPlanHandler(pipe).Handle)
	mux.HandleFunc("POST /v1/execute", NewExecuteHandler(exec).Handle)
	mux.HandleFunc("POST /v1/replan-with-hint", NewReplanHandler(prov).Handle)

	return &env{
		mux:         mux,
		downloadDir: downloadDir,
		targetDir:   targetDir,
		provider:    prov,
	}
}

func postJSON(t *testing.T, e *env, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
}

func TestPlanHandler_OKContract(t *testing.T) {
	prov := mock.NewProvider()
	e := newTestEnv(t, prov)

	// A pure-extension book hits the offline matcher: no LLM is involved at
	// all, so the handler is tested in complete isolation.
	rec := postJSON(t, e, "/v1/plan", model.APIPlanRequest{
		Dir:      "hash1",
		Files:    []string{"mybook.epub"},
		Metadata: map[string]interface{}{"title": "My Book"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	decodeBody(t, rec, &raw)

	// Contract: "error" must be present and null.
	errField, ok := raw["error"]
	if !ok || errField != nil {
		t.Fatalf("error must be null in normal planning, got %v", errField)
	}
	plan, ok := raw["plan"].([]interface{})
	if !ok || len(plan) != 1 {
		t.Fatalf("unexpected plan: %v", raw["plan"])
	}
	action := plan[0].(map[string]interface{})
	if action["file"] != "mybook.epub" || action["action"] != "move" || action["target"] != "book/mybook.epub" {
		t.Fatalf("unexpected action: %v", action)
	}
}

func TestPlanHandler_FatalError500(t *testing.T) {
	prov := mock.NewProvider()
	prov.SetDefaultResponse(nil, errors.New("categorizer offline"))
	e := newTestEnv(t, prov)

	// Unknown extension forces the Stage 1 LLM fallback, which fails fatally.
	rec := postJSON(t, e, "/v1/plan", model.APIPlanRequest{
		Dir:   "hash2",
		Files: []string{"mystery.bin"},
	})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("fatal pipeline error must map to 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if resp.Error == nil || !strings.Contains(*resp.Error, "categorizer offline") {
		t.Fatalf("500 body must carry the error, got %+v", resp.Error)
	}
}

func TestPlanHandler_UnknownCategoryEmptyPlan200(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "media categorization assistant",
		Response:      `{"category":"unknown","reason":"junk","entities":{}}`,
	})
	e := newTestEnv(t, prov)

	rec := postJSON(t, e, "/v1/plan", model.APIPlanRequest{
		Dir:   "hash3",
		Files: []string{"mystery.bin"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown category is normal planning, expected 200, got %d", rec.Code)
	}
	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if resp.Error != nil {
		t.Fatalf("error must stay null, got %s", *resp.Error)
	}
	if len(resp.Plan) != 0 {
		t.Fatalf("unknown category must return an empty plan, got %+v", resp.Plan)
	}
}

func TestExecuteHandler_Success200AndArchive(t *testing.T) {
	prov := mock.NewProvider()
	e := newTestEnv(t, prov)

	if err := os.MkdirAll(filepath.Join(e.downloadDir, "d1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.downloadDir, "d1", "movie.mkv"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postJSON(t, e, "/v1/execute", model.APIExecuteRequest{
		Dir: "d1",
		Plan: []model.PlanAction{
			{File: "movie.mkv", Action: "move", Target: strPtr("movie/Others/M (2000)/M (2000).mkv")},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp model.ExecuteResponse
	decodeBody(t, rec, &resp)
	if len(resp.FailedMove) != 0 {
		t.Fatalf("expected empty failed_move, got %+v", resp.FailedMove)
	}
	if _, err := os.Stat(filepath.Join(e.targetDir, "movie", "Others", "M (2000)", "M (2000).mkv")); err != nil {
		t.Fatalf("file must be moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.downloadDir, "archive", "d1")); err != nil {
		t.Fatalf("source dir must be archived on full success: %v", err)
	}
}

func TestExecuteHandler_PartialFailure400(t *testing.T) {
	prov := mock.NewProvider()
	e := newTestEnv(t, prov)

	if err := os.MkdirAll(filepath.Join(e.downloadDir, "d2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.downloadDir, "d2", "good.mkv"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := postJSON(t, e, "/v1/execute", model.APIExecuteRequest{
		Dir: "d2",
		Plan: []model.PlanAction{
			{File: "missing.mkv", Action: "move", Target: strPtr("movie/Others/M (2000)/M (2000).mkv")},
			{File: "good.mkv", Action: "move", Target: strPtr("movie/Others/M (2000)/M (2000) part.2.mkv")},
		},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("aggregated failure must map to 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.ExecuteResponse
	decodeBody(t, rec, &resp)
	if len(resp.FailedMove) != 1 {
		t.Fatalf("expected one failed_move entry, got %+v", resp.FailedMove)
	}
	failed := resp.FailedMove[0]
	if failed.File != "missing.mkv" || failed.Reason != "file not found" {
		t.Fatalf("unexpected failed entry: %+v", failed)
	}
	// The legal move must still have been executed (L11).
	if _, err := os.Stat(filepath.Join(e.targetDir, "movie", "Others", "M (2000)", "M (2000) part.2.mkv")); err != nil {
		t.Fatalf("legal move must proceed: %v", err)
	}
	// No archive on failure.
	if _, err := os.Stat(filepath.Join(e.downloadDir, "archive", "d2")); !os.IsNotExist(err) {
		t.Fatalf("source dir must not be archived on failure, stat err: %v", err)
	}
}

func TestExecuteHandler_MethodNotAllowed(t *testing.T) {
	prov := mock.NewProvider()
	e := newTestEnv(t, prov)

	req := httptest.NewRequest(http.MethodGet, "/v1/execute", nil)
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /v1/execute must be rejected, got %d", rec.Code)
	}
}

func TestReplanHandler_TVDomainRouting(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "revises a TV series file organization plan",
		Response: `{"plan":[
			{"file":"Show S01E01.mkv","action":"move","target":"tv_series/Others/Show (2020)/Season 02/Show (2020) S02E01.mkv"},
			{"file":"Show S01E02.mkv","action":"move","target":"../escape.mkv"}]}`,
	})
	e := newTestEnv(t, prov)

	prevTarget := "tv_series/Others/Show (2020)/Season 01/Show (2020) S01E01.mkv"
	rec := postJSON(t, e, "/v1/replan-with-hint", model.APIReplanRequest{
		Files:    []string{"Show S01E01.mkv", "Show S01E02.mkv"},
		Metadata: map[string]interface{}{"title": "Show"},
		PreviousResponse: model.PlanResponse{Plan: []model.PlanAction{
			{File: "Show S01E01.mkv", Action: "move", Target: &prevTarget},
		}},
		UserHint: "these are actually season 2 episodes",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if resp.Error != nil {
		t.Fatalf("error must stay null, got %s", *resp.Error)
	}
	if len(resp.Plan) != 2 {
		t.Fatalf("every file must appear exactly once, got %+v", resp.Plan)
	}

	// L14: the domain (TV) replan prompt must have been used.
	calls := prov.Calls()
	if len(calls) != 1 {
		t.Fatalf("single replan must issue exactly one LLM call, got %d", len(calls))
	}
	if !strings.Contains(calls[0].Prompt, "revises a TV series file organization plan") {
		t.Fatalf("tv domain prompt must be used, got: %s", calls[0].Prompt)
	}
	if !strings.Contains(calls[0].Prompt, "these are actually season 2 episodes") {
		t.Fatalf("user hint must be injected into the prompt")
	}

	byFile := map[string]model.PlanAction{}
	for _, a := range resp.Plan {
		byFile[a.File] = a
	}
	if a := byFile["Show S01E01.mkv"]; a.Action != "move" || a.Target == nil ||
		*a.Target != "tv_series/Others/Show (2020)/Season 02/Show (2020) S02E01.mkv" {
		t.Fatalf("unexpected replanned action: %+v", a)
	}
	// Stage 4 security bottom line still applies to replans: traversal forced skip.
	if a := byFile["Show S01E02.mkv"]; a.Action != "skip" || a.Target != nil {
		t.Fatalf("traversal target must be sanitized to skip, got %+v", a)
	}
}

func TestReplanHandler_EmptyPlanFallsBackToGenericPrompt(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "revises file organization plans based on user feedback",
		Response: `{"plan":[{"file":"movie.mkv","action":"move",
			"target":"movie/Others/Movie (2000)/Movie (2000).mkv"}]}`,
	})
	e := newTestEnv(t, prov)

	rec := postJSON(t, e, "/v1/replan-with-hint", model.APIReplanRequest{
		Files:            []string{"movie.mkv"},
		Metadata:         map[string]interface{}{"title": "Movie"},
		PreviousResponse: model.PlanResponse{Plan: []model.PlanAction{}},
		UserHint:         "the year should be 2000, not 2024",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	calls := prov.Calls()
	if len(calls) != 1 || !strings.Contains(calls[0].Prompt, "revises file organization plans based on user feedback") {
		t.Fatalf("empty previous plan must fall back to the generic replan prompt, calls: %+v", calls)
	}

	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if resp.Error != nil || len(resp.Plan) != 1 || resp.Plan[0].Action != "move" {
		t.Fatalf("unexpected replan response: %+v", resp)
	}
}

func TestReplanHandler_UnknownRootFallsBackToGenericPrompt(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "revises file organization plans based on user feedback",
		Response:      `{"plan":[{"file":"a.mkv","action":"skip"}]}`,
	})
	e := newTestEnv(t, prov)

	unknownRoot := "weird_root/a.mkv"
	rec := postJSON(t, e, "/v1/replan-with-hint", model.APIReplanRequest{
		Files: []string{"a.mkv"},
		PreviousResponse: model.PlanResponse{Plan: []model.PlanAction{
			{File: "a.mkv", Action: "move", Target: &unknownRoot},
		}},
		UserHint: "unknown domain",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	calls := prov.Calls()
	if len(calls) != 1 || !strings.Contains(calls[0].Prompt, "revises file organization plans based on user feedback") {
		t.Fatalf("uninferable root must fall back to the generic replan prompt, calls: %+v", calls)
	}
	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if len(resp.Plan) != 1 || resp.Plan[0].Action != "skip" || resp.Plan[0].Target != nil {
		t.Fatalf("unexpected replan response: %+v", resp.Plan)
	}
}

func TestReplanHandler_LLMFailure500(t *testing.T) {
	prov := mock.NewProvider()
	prov.SetDefaultResponse(nil, errors.New("replanner offline"))
	e := newTestEnv(t, prov)

	rec := postJSON(t, e, "/v1/replan-with-hint", model.APIReplanRequest{
		Files:            []string{"a.mkv"},
		PreviousResponse: model.PlanResponse{Plan: []model.PlanAction{}},
		UserHint:         "fix it",
	})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("fatal replan error must map to 500, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp model.PlanResponse
	decodeBody(t, rec, &resp)
	if resp.Error == nil || !strings.Contains(*resp.Error, "replanner offline") {
		t.Fatalf("500 body must carry the error, got %+v", resp.Error)
	}
}

func TestReplanHandler_InvalidBody400(t *testing.T) {
	prov := mock.NewProvider()
	e := newTestEnv(t, prov)

	req := httptest.NewRequest(http.MethodPost, "/v1/replan-with-hint", strings.NewReader("{invalid"))
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON must map to 400, got %d", rec.Code)
	}
}

// Compile-time interface guards.
var _ ai.Provider = (*mock.Provider)(nil)

func strPtr(s string) *string { return &s }
