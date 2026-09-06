package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/ai/gemini"
	"github.com/autoget-project/organizer/internal/ai/grok"
	"github.com/autoget-project/organizer/internal/model"
	"github.com/autoget-project/organizer/internal/testutil"
)

func TestE2E_LiveProviderBusinessLoop(t *testing.T) {
	// Keys and per-provider models come from the gitignored .env.e2e at the
	// repo root; variables already exported in the shell keep precedence.
	// XAI_MODEL and GEMINI_MODEL are independent, so grok and gemini can be
	// exercised in a single run.
	//
	// LoadEnvFile relies on t.Setenv, which is incompatible with a
	// top-level t.Parallel; the per-provider subtests below parallelize.
	testutil.LoadEnvFile(t, filepath.Join("..", "..", ".env.e2e"))

	targets := []struct {
		modelEnv string
		keyEnv   string
		newProv  func(t *testing.T, modelName, apiKey string) ai.Provider
	}{
		{"XAI_MODEL", "XAI_API_KEY", func(t *testing.T, m, k string) ai.Provider {
			t.Helper()
			return grok.NewProvider(k, ai.WithModel(m))
		}},
		{"GEMINI_MODEL", "GEMINI_API_KEY", func(t *testing.T, m, k string) ai.Provider {
			t.Helper()
			prov, err := gemini.NewProvider(k, ai.WithModel(m))
			require.NoError(t, err)
			return prov
		}},
	}

	configured := false
	for _, target := range targets {
		modelName := strings.TrimSpace(os.Getenv(target.modelEnv))
		if modelName == "" {
			continue
		}
		configured = true
		t.Run(modelName, func(t *testing.T) {
			t.Parallel()

			apiKey := os.Getenv(target.keyEnv)
			if apiKey == "" {
				t.Skipf("Skipping live loop: %s is set but %s is missing", target.modelEnv, target.keyEnv)
			}
			runLiveBusinessLoop(t, target.newProv(t, modelName, apiKey))
		})
	}
	if !configured {
		t.Skip("Skipping live loop: neither XAI_MODEL nor GEMINI_MODEL is set")
	}
}

// runLiveBusinessLoop drives the real provider through the full business
// closed loop: deterministic offline plan, LLM movie plan, execute, archive.
func runLiveBusinessLoop(t *testing.T, prov ai.Provider) {
	t.Helper()

	s := newSandbox(t, prov)

	// Deterministic offline rule path (pure eBook): no LLM involved, the plan
	// contract must hold with any provider implementation.
	s.seedDownloadFile(t, "livedl", "MyAwesomeBook.epub", "book data")
	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "livedl",
		Files: []string{"MyAwesomeBook.epub"},
	})
	require.Equal(t, http.StatusOK, code, body)
	var bookPlan model.PlanResponse
	decodeBody(t, code, body, &bookPlan)
	require.Nil(t, bookPlan.Error)
	require.Len(t, bookPlan.Plan, 1)
	require.Equal(t, "move", bookPlan.Plan[0].Action)
	require.NotNil(t, bookPlan.Plan[0].Target)
	assert.Equal(t, "book/MyAwesomeBook.epub", *bookPlan.Plan[0].Target)

	// Real LLM closed loop on a messy movie release: assert structural
	// invariants (never exact LLM wording), then execute and verify delivery.
	const video = "Inception.2010.1080p.BluRay.x264.mkv"
	s.seedDownloadFile(t, "livedl-movie", video, "mkvdata")
	code, body = s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "livedl-movie",
		Files: []string{video},
	})
	require.Equal(t, http.StatusOK, code, body)
	var moviePlan model.PlanResponse
	decodeBody(t, code, body, &moviePlan)
	require.Nil(t, moviePlan.Error, "live movie plan error must be null")

	moved := 0
	for _, a := range moviePlan.Plan {
		require.Contains(t, []string{"move", "skip"}, a.Action, "invalid action in live plan: %+v", a)
		if a.Action == "move" {
			moved++
			require.NotNil(t, a.Target, "move without target in live plan: %+v", a)
			root := strings.SplitN(*a.Target, "/", 2)[0]
			assert.Contains(t, allTargetDirNames(), root, "live plan target must stay inside a known root dir")
		}
	}
	assert.Equal(t, 1, moved, "exactly one main feature must be moved in %+v", moviePlan.Plan)

	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "livedl-movie",
		Plan: moviePlan.Plan,
	})
	require.Equal(t, http.StatusOK, code, body)
	var exec model.ExecuteResponse
	decodeBody(t, code, body, &exec)
	assert.Empty(t, exec.FailedMove)
	assert.DirExists(t, filepath.Join(s.downloadDir, "archive", "livedl-movie"))
}

// allTargetDirNames returns the known target root dir names.
func allTargetDirNames() []string {
	names := make([]string, 0, len(model.AllTargetDirs))
	for _, d := range model.AllTargetDirs {
		names = append(names, string(d))
	}
	return names
}
