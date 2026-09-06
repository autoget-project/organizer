package e2e

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

// TestE2E_ReplanWithHintLifecycle drives /v1/plan -> /v1/replan-with-hint ->
// /v1/execute: the user hint is applied on top of the previous plan without
// re-running Stage 1/2, and executing the replanned result delivers the file
// into TARGET_DIR and archives the source directory.
func TestE2E_ReplanWithHintLifecycle(t *testing.T) {
	t.Parallel()

	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"movie","reason":"feature film","entities":{}}`})
	prov.AddRule(mock.Rule{PromptPattern: patMovie, Response: `{"plan":[
		{"file":"movie.mkv","action":"move","target":"movie/English/Wrong Name (2020)/Wrong Name (2020).mkv"}
	]}`})
	// The previous plan's first move target starts with "movie", so the movie
	// domain replan prompt is used (L14 single-shot domain injection).
	prov.AddRule(mock.Rule{PromptPattern: "revises a movie file organization plan", Response: `{"plan":[
		{"file":"movie.mkv","action":"move","target":"movie/English/The Correct Name (2020)/The Correct Name (2020).mkv"}
	]}`})

	s.seedDownloadFile(t, "replandl", "movie.mkv", "mkvdata")

	// Initial plan.
	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:      "replandl",
		Files:    []string{"movie.mkv"},
		Metadata: map[string]interface{}{"title": "Wrong Name", "year": 2020},
	})
	require.Equal(t, http.StatusOK, code, body)
	var initial model.PlanResponse
	decodeBody(t, code, body, &initial)
	require.Nil(t, initial.Error)
	require.Len(t, initial.Plan, 1)

	// Replan with a user hint: stage 1/2 are never re-run.
	code, body = s.postJSON(t, "/v1/replan-with-hint", model.APIReplanRequest{
		Files:            []string{"movie.mkv"},
		Metadata:         map[string]interface{}{"title": "Wrong Name", "year": 2020},
		PreviousResponse: initial,
		UserHint:         "the movie name is wrong, it should be The Correct Name",
	})
	require.Equal(t, http.StatusOK, code, body)
	var replanned model.PlanResponse
	decodeBody(t, code, body, &replanned)
	require.Nil(t, replanned.Error)
	require.Len(t, replanned.Plan, 1)
	require.NotNil(t, replanned.Plan[0].Target, "user hint must be applied")
	assert.Equal(t, "movie/English/The Correct Name (2020)/The Correct Name (2020).mkv", *replanned.Plan[0].Target)

	// Execute the replanned plan: file lands and source dir is archived.
	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "replandl",
		Plan: replanned.Plan,
	})
	require.Equal(t, http.StatusOK, code, body)
	assert.FileExists(t, filepath.Join(s.targetDir, "movie", "English", "The Correct Name (2020)", "The Correct Name (2020).mkv"))
	assert.DirExists(t, filepath.Join(s.downloadDir, "archive", "replandl"))
}
