// Package e2e hosts the full-scenario end-to-end test suite. It drives the
// real HTTP surface (/v1/plan, /v1/execute, /v1/replan-with-hint) inside an
// isolated sandbox (t.TempDir) and covers one agent per file: the Stage 1
// categorizer, the Stage 3 movers (simple / tv_series / movie / bango_porn /
// subtitle), the executor and the replan agent, so the planning & execution
// behavior of every pipeline stage is verified over the wire.
//
// Offline CI: LLM-dependent paths run against the deterministic mock provider
// (rule-based structured responses identical to what the real LLM must
// return), so `just test` / `just test-e2e` never break offline builds.
// Live CI: TestE2E_LiveProviderBusinessLoop in live_test.go loads the
// gitignored .env.e2e (XAI_API_KEY / GEMINI_API_KEY plus per-provider
// XAI_MODEL / GEMINI_MODEL, already-exported variables win) - it gracefully
// skips without configuration and runs the real business closed loop for
// every configured provider in parallel subtests.
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/handler"
	"github.com/autoget-project/organizer/internal/model"
	"github.com/autoget-project/organizer/internal/pipeline"
	stage2enricher "github.com/autoget-project/organizer/internal/pipeline/stage2_enricher"
	"github.com/autoget-project/organizer/internal/ptr"
	"github.com/autoget-project/organizer/internal/service"
)

// sandbox is an isolated E2E environment: a live HTTP server exposing the
// REST routes, backed by temporary DOWNLOAD_COMPLETED_DIR / TARGET_DIR roots
// with every TargetDir enum subdirectory pre-created.
type sandbox struct {
	server      *httptest.Server
	downloadDir string
	targetDir   string
}

// newSandbox wires the full server around the given provider inside a fresh
// sandbox. All model.TargetDirs subdirectories are pre-created so a
// StartupCheck-equivalent validation would never report missing dirs.
func newSandbox(t *testing.T, prov ai.Provider) *sandbox {
	t.Helper()

	downloadDir := t.TempDir()
	targetDir := t.TempDir()

	for _, d := range model.AllTargetDirs {
		require.NoError(t, os.MkdirAll(filepath.Join(targetDir, string(d)), 0o755),
			"pre-create target subdirectory %s", d)
	}

	// Stage 2 runs with nil metadata sources: enrichment degrades gracefully
	// (M6) exactly like the offline production degradation path.
	pipe := pipeline.NewPipeline(prov, stage2enricher.NewEnricher(nil, nil, nil, nil), downloadDir)
	exec := service.NewExecutor(downloadDir, targetDir)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/plan", handler.NewPlanHandler(pipe).Handle)
	mux.HandleFunc("POST /v1/execute", handler.NewExecuteHandler(exec).Handle)
	mux.HandleFunc("POST /v1/replan-with-hint", handler.NewReplanHandler(prov).Handle)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &sandbox{
		server:      server,
		downloadDir: downloadDir,
		targetDir:   targetDir,
	}
}

// newMockSandbox creates a sandbox around a fresh rule-based mock provider.
func newMockSandbox(t *testing.T) (*sandbox, *mock.Provider) {
	t.Helper()

	prov := mock.NewProvider()
	return newSandbox(t, prov), prov
}

// seedDownloadFile materializes a file inside DOWNLOAD_COMPLETED_DIR/{dir}.
func (s *sandbox) seedDownloadFile(t *testing.T, dir, file, content string) {
	t.Helper()

	full := filepath.Join(s.downloadDir, dir, file)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644), "seed file %s", full)
}

// postJSON performs a real HTTP round trip against the sandbox server and
// returns the status code plus raw body.
func (s *sandbox) postJSON(t *testing.T, path string, payload interface{}) (int, string) {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(s.server.URL+path, "application/json", bytes.NewReader(body))
	require.NoError(t, err, "POST %s", path)
	t.Cleanup(func() { _ = resp.Body.Close() })

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err, "read response body")
	return resp.StatusCode, buf.String()
}

func decodeBody(t *testing.T, code int, body string, v interface{}) {
	t.Helper()

	require.NoError(t, json.Unmarshal([]byte(body), v), "decode response (status %d) %q", code, body)
}

// assertPlanContract checks the API response contract: HTTP 200, error null,
// every input file appears exactly once, and each action matches the
// expected mapping (file -> action/target). Skip actions must serialize a
// null target.
func assertPlanContract(t *testing.T, code int, body string, want map[string]model.PlanAction) {
	t.Helper()

	require.Equal(t, http.StatusOK, code, "plan must succeed: %s", body)
	var resp model.PlanResponse
	decodeBody(t, code, body, &resp)
	assert.Nil(t, resp.Error, "error must stay null in normal planning")
	require.Len(t, resp.Plan, len(want), "plan must cover every file exactly once: %+v", resp.Plan)

	got := make(map[string]model.PlanAction, len(resp.Plan))
	for _, a := range resp.Plan {
		got[a.File] = a
	}

	files := make([]string, 0, len(want))
	for f := range want {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, f := range files {
		w := want[f]
		g, ok := got[f]
		require.True(t, ok, "file %q missing from plan: %+v", f, resp.Plan)
		assert.Equal(t, w.Action, g.Action, "file %q action", f)
		if w.Action == "move" {
			require.NotNil(t, g.Target, "file %q: move action must carry a target", f)
			assert.Equal(t, *w.Target, *g.Target, "file %q target", f)
		} else {
			assert.Nil(t, g.Target, "file %q: skip action must serialize target null", f)
		}
	}
}

// wantMove / wantSkip build the expected action entries.
func wantMove(target string) model.PlanAction {
	return model.PlanAction{Action: "move", Target: ptr.Str(target)}
}
func wantSkip() model.PlanAction { return model.PlanAction{Action: "skip"} }

// LLM prompt fingerprints used by mock rules (must mirror production prompts).
const (
	patClassifier = "media categorization assistant"
	patTVPlanner  = "organizes TV series downloads"
	patMovie      = "organizes movie downloads"
	patBango      = "specialized file mover for bango porn videos"
	patSubtitle   = "organizes subtitle files"
	patReplan     = "revises file organization plans based on user feedback"
)
