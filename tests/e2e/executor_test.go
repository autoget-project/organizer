package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
	"organizer/internal/ptr"
)

// TestE2E_ExecutePlan covers the physical execution contract: a move lands
// the file under TARGET_DIR and removes it from the download directory.
func TestE2E_ExecutePlan(t *testing.T) {
	t.Parallel()

	s, _ := newMockSandbox(t)

	s.seedDownloadFile(t, "subfolder", "test_file.txt", "test content")

	code, body := s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir: "subfolder",
		Plan: []model.PlanAction{
			{File: "test_file.txt", Action: "move", Target: ptr.Str("documents/test_file.txt")},
		},
	})
	require.Equal(t, http.StatusOK, code, body)

	var resp model.ExecuteResponse
	decodeBody(t, code, body, &resp)
	assert.Empty(t, resp.FailedMove)

	assert.FileExists(t, filepath.Join(s.targetDir, "documents", "test_file.txt"))
	assert.NoFileExists(t, filepath.Join(s.downloadDir, "subfolder", "test_file.txt"))
}

// TestE2E_ExecutePlanFailed covers failure aggregation: a missing source file
// yields a failed_move reason "file not found" without creating the target.
func TestE2E_ExecutePlanFailed(t *testing.T) {
	t.Parallel()

	s, _ := newMockSandbox(t)

	require.NoError(t, os.MkdirAll(filepath.Join(s.downloadDir, "subfolder"), 0o755))

	code, body := s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir: "subfolder",
		Plan: []model.PlanAction{
			{File: "non_existent_file.txt", Action: "move", Target: ptr.Str("documents/non_existent_file.txt")},
		},
	})
	require.Equal(t, http.StatusBadRequest, code, body)

	var resp model.ExecuteResponse
	decodeBody(t, code, body, &resp)
	require.NotEmpty(t, resp.FailedMove)
	assert.Contains(t, resp.FailedMove[0].Reason, "file not found")

	assert.NoFileExists(t, filepath.Join(s.targetDir, "documents", "non_existent_file.txt"))
}

// TestE2E_DirectoryLevelAtomicMoveLifecycle covers multi-file directory-level
// atomic moves (H2) for book / audio_book content inside a hash wrapper
// directory: plan -> execute -> filesystem verification -> archive/{dir}.
func TestE2E_DirectoryLevelAtomicMoveLifecycle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		dir    string
		files  []string
		rules  []mock.Rule
		target string // planned whole-directory target
	}{
		{
			// Multi-file book: pure eBook extensions hit the offline rule, no LLM.
			name:   "book_hash_dir",
			dir:    "bookdl",
			files:  []string{"b00khash/chapter1.pdf", "b00khash/notes.txt"},
			target: "book/b00khash",
		},
		{
			// Multi-file audio book: pure audio triggers the M8b LLM
			// disambiguation, which classifies chapter-named MP3s as audio_book.
			name:  "audio_book_hash_dir",
			dir:   "abdl",
			files: []string{"abhash/Chapter 01.mp3", "abhash/Chapter 02.mp3"},
			rules: []mock.Rule{
				{PromptPattern: patClassifier, Response: `{"category":"audio_book","reason":"chapter-named narration","entities":{}}`},
			},
			target: "audio_book/abhash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, prov := newMockSandbox(t)
			for _, r := range tc.rules {
				prov.AddRule(r)
			}
			for _, f := range tc.files {
				s.seedDownloadFile(t, tc.dir, f, "data")
			}

			// Plan: the whole hash wrapper directory is moved (branch 2).
			hashDir := strings.SplitN(tc.files[0], "/", 2)[0]
			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:   tc.dir,
				Files: tc.files,
			})
			assertPlanContract(t, code, body, map[string]model.PlanAction{
				hashDir: wantMove(tc.target),
			})

			// Execute: directory-level atomic move lands in TARGET_DIR.
			code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
				Dir:  tc.dir,
				Plan: []model.PlanAction{{File: hashDir, Action: "move", Target: ptr.Str(tc.target)}},
			})
			require.Equal(t, http.StatusOK, code, body)
			var exec model.ExecuteResponse
			decodeBody(t, code, body, &exec)
			assert.Empty(t, exec.FailedMove)

			// Every seeded file must have landed under TARGET_DIR.
			for _, f := range tc.files {
				rel := strings.SplitN(f, "/", 2)[1]
				assert.FileExists(t, filepath.Join(s.targetDir, tc.target, rel))
			}

			// Source directory must be archived to archive/{dir} and be gone.
			assert.DirExists(t, filepath.Join(s.downloadDir, "archive", tc.dir))
			assert.NoDirExists(t, filepath.Join(s.downloadDir, tc.dir))
		})
	}
}

// TestE2E_FullLifecyclePlanExecuteArchive covers the physical execution
// lifecycle closed loop: /v1/plan -> /v1/execute -> file landed in TARGET_DIR
// -> source dir archived to archive/{dir}.
func TestE2E_FullLifecyclePlanExecuteArchive(t *testing.T) {
	t.Parallel()

	const noiseDir = "[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi"
	video := noiseDir + "/The.Matrix.1999.mkv"
	sample := noiseDir + "/The.Matrix.1999.sample.mkv"

	s, prov := newMockSandbox(t)
	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"movie","reason":"feature film","entities":{"clean_title":"The Matrix","year":1999}}`})
	prov.AddRule(mock.Rule{PromptPattern: patMovie, Response: `{"plan":[
		{"file":"` + video + `","action":"move","target":"movie/English/The Matrix (1999)/The Matrix (1999).mkv"},
		{"file":"` + sample + `","action":"skip"}
	]}`})

	s.seedDownloadFile(t, "matrixdl", video, "mkvdata")
	s.seedDownloadFile(t, "matrixdl", sample, "sampledata")

	// Step 1: plan.
	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:      "matrixdl",
		Files:    []string{video, sample},
		Metadata: map[string]interface{}{"title": "The Matrix", "year": 1999},
	})
	require.Equal(t, http.StatusOK, code, body)
	var plan model.PlanResponse
	decodeBody(t, code, body, &plan)
	require.Nil(t, plan.Error)
	require.Len(t, plan.Plan, 2)

	// Step 2: execute the planned actions verbatim.
	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "matrixdl",
		Plan: plan.Plan,
	})
	require.Equal(t, http.StatusOK, code, body)
	var exec model.ExecuteResponse
	decodeBody(t, code, body, &exec)
	assert.Empty(t, exec.FailedMove)

	// Step 3: the main feature must be in TARGET_DIR, the sample dropped.
	assert.FileExists(t, filepath.Join(s.targetDir, "movie", "English", "The Matrix (1999)", "The Matrix (1999).mkv"))
	assert.NoFileExists(t, filepath.Join(s.targetDir, "movie", "English", "The Matrix (1999)", "The Matrix (1999).sample.mkv"))

	// Step 4: source directory archived to archive/{dir}.
	assert.DirExists(t, filepath.Join(s.downloadDir, "archive", "matrixdl"))
	assert.NoDirExists(t, filepath.Join(s.downloadDir, "matrixdl"))
}
