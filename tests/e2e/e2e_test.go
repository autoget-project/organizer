// Package e2e hosts the full-scenario end-to-end test suite. It drives the
// real HTTP surface (/v1/plan, /v1/execute, /v1/replan-with-hint) inside an
// isolated sandbox (t.TempDir) and ports every legacy Python test case from
// archived/app/main_test.py, archived/app/agents/mover/*_test.py and
// archived/app/agents/categorizer/*_test.py into table-driven Go tests so the
// planning & execution behavior stays 100% aligned with the legacy system.
//
// Offline CI: LLM-dependent paths run against the deterministic mock provider
// (rule-based structured responses identical to what the real LLM must
// return), so `just test` / `just test-e2e` never break offline builds.
// Live CI: TestE2E_LiveProviderBusinessLoop loads the gitignored .env.e2e
// (XAI_API_KEY / GEMINI_API_KEY plus per-provider XAI_MODEL / GEMINI_MODEL,
// already-exported variables win) - it gracefully skips without configuration
// and runs the real business closed loop for every configured provider in
// parallel subtests.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"organizer/internal/ai"
	"organizer/internal/ai/gemini"
	"organizer/internal/ai/grok"
	"organizer/internal/ai/mock"
	"organizer/internal/handler"
	"organizer/internal/model"
	"organizer/internal/pipeline"
	"organizer/internal/pipeline/stage2_enricher"
	"organizer/internal/ptr"
	"organizer/internal/service"
	"organizer/internal/testutil"
)

// sandbox is an isolated E2E environment: a live HTTP server exposing the
// legacy-compatible REST routes, backed by temporary DOWNLOAD_COMPLETED_DIR /
// TARGET_DIR roots with every TargetDir enum subdirectory pre-created.
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
		if err := os.MkdirAll(filepath.Join(targetDir, string(d)), 0o755); err != nil {
			t.Fatalf("pre-create target subdirectory %s: %v", d, err)
		}
	}

	// Stage 2 runs with a nil MCP client: enrichment degrades gracefully (M6)
	// exactly like the offline production degradation path.
	pipe := pipeline.NewPipeline(prov, stage2enricher.NewEnricher(nil, nil, nil), downloadDir)
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
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("seed parent dir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("seed file %s: %v", full, err)
	}
}

// postJSON performs a real HTTP round trip against the sandbox server and
// returns the status code plus raw body.
func (s *sandbox) postJSON(t *testing.T, path string, payload interface{}) (int, string) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resp, err := http.Post(s.server.URL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, buf.String()
}

func decodeBody(t *testing.T, code int, body string, v interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("decode response (status %d) %q: %v", code, body, err)
	}
}

// assertPlanContract checks the legacy response contract: HTTP 200, error
// null, every input file appears exactly once, and each action matches the
// expected mapping (file -> action/target). Skip actions must serialize a
// null target.
func assertPlanContract(t *testing.T, code int, body string, want map[string]model.PlanAction) {
	t.Helper()
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}
	var resp model.PlanResponse
	decodeBody(t, code, body, &resp)
	if resp.Error != nil {
		t.Fatalf("error must stay null in normal planning, got %q", *resp.Error)
	}
	if len(resp.Plan) != len(want) {
		t.Fatalf("plan must cover every file exactly once: got %d actions (%+v), want %d", len(resp.Plan), resp.Plan, len(want))
	}
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
		if !ok {
			t.Fatalf("file %q missing from plan: %+v", f, resp.Plan)
		}
		if g.Action != w.Action {
			t.Fatalf("file %q: got action %q, want %q", f, g.Action, w.Action)
		}
		if w.Action == "move" {
			if g.Target == nil {
				t.Fatalf("file %q: move action must carry a target, got null", f)
			}
			if *g.Target != *w.Target {
				t.Fatalf("file %q: got target %q, want %q", f, *g.Target, *w.Target)
			}
		} else if g.Target != nil {
			t.Fatalf("file %q: skip action must serialize target null, got %q", f, *g.Target)
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

// ---------------------------------------------------------------------------
// Scenario 1 (a): legacy archived/app/main_test.py ports.
// ---------------------------------------------------------------------------

// TestE2E_LegacyPythonExecutePlan ports archived/app/main_test.py
// test_execute_plan: a physical move lands the file under TARGET_DIR and
// removes it from the download directory.
func TestE2E_LegacyPythonExecutePlan(t *testing.T) {
	s, _ := newMockSandbox(t)

	s.seedDownloadFile(t, "subfolder", "test_file.txt", "test content")

	code, body := s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir: "subfolder",
		Plan: []model.PlanAction{
			{File: "test_file.txt", Action: "move", Target: ptr.Str("documents/test_file.txt")},
		},
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", code, body)
	}
	var resp model.ExecuteResponse
	decodeBody(t, code, body, &resp)
	if len(resp.FailedMove) != 0 {
		t.Fatalf("expected empty failed_move, got %+v", resp.FailedMove)
	}
	if _, err := os.Stat(filepath.Join(s.targetDir, "documents", "test_file.txt")); err != nil {
		t.Fatalf("target file must exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.downloadDir, "subfolder", "test_file.txt")); !os.IsNotExist(err) {
		t.Fatalf("source file must be gone, stat err: %v", err)
	}
}

// TestE2E_LegacyPythonExecutePlanFailed ports archived/app/main_test.py
// test_execute_plan_failed: a missing source file yields an aggregated
// failed_move reason "file not found" without creating the target.
func TestE2E_LegacyPythonExecutePlanFailed(t *testing.T) {
	s, _ := newMockSandbox(t)

	if err := os.MkdirAll(filepath.Join(s.downloadDir, "subfolder"), 0o755); err != nil {
		t.Fatal(err)
	}

	code, body := s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir: "subfolder",
		Plan: []model.PlanAction{
			{File: "non_existent_file.txt", Action: "move", Target: ptr.Str("documents/non_existent_file.txt")},
		},
	})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", code, body)
	}
	var resp model.ExecuteResponse
	decodeBody(t, code, body, &resp)
	if len(resp.FailedMove) == 0 || !strings.Contains(resp.FailedMove[0].Reason, "file not found") {
		t.Fatalf("expected 'file not found' reason, got %+v", resp.FailedMove)
	}
	if _, err := os.Stat(filepath.Join(s.targetDir, "documents", "non_existent_file.txt")); !os.IsNotExist(err) {
		t.Fatalf("target must not be created, stat err: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 1 (b): legacy archived/app/agents/mover/simple_mover_test.py ports
// (three-branch archiving strategy), driven through POST /v1/plan.
// ---------------------------------------------------------------------------

func TestE2E_LegacyPythonSimpleMoverBranches(t *testing.T) {
	cases := []struct {
		name     string
		files    []string
		metadata map[string]interface{}
		want     map[string]model.PlanAction
	}{
		{
			// Port of test_simple_move_plan_single_file (branch 1).
			name:     "single_file",
			files:    []string{"torrent_hash/movie.mp4"},
			metadata: map[string]interface{}{"organizer_category": "music_video"},
			want: map[string]model.PlanAction{
				"torrent_hash/movie.mp4": wantMove("music_video/movie.mp4"),
			},
		},
		{
			// Port of test_simple_move_plan_multiple_files_under_hash_dir (branch 2).
			name:     "multiple_files_under_hash_dir",
			files:    []string{"torrent_hash/episode1.mp4", "torrent_hash/episode2.mp4"},
			metadata: map[string]interface{}{"organizer_category": "music_video"},
			want: map[string]model.PlanAction{
				"torrent_hash": wantMove("music_video/torrent_hash"),
			},
		},
		{
			// Port of test_simple_move_plan_multiple_files_in_subdirs (branch 3,
			// hash layer stripped). Pure eBook extensions hit the offline Stage 1
			// book rule, so no LLM is involved at all.
			name: "multiple_files_in_subdirs",
			files: []string{
				"torrent_hash/chapter1/page1.pdf",
				"torrent_hash/chapter1/page2.pdf",
				"torrent_hash/chapter2/page1.pdf",
			},
			metadata: nil,
			want: map[string]model.PlanAction{
				"torrent_hash/chapter1": wantMove("book/chapter1"),
				"torrent_hash/chapter2": wantMove("book/chapter2"),
			},
		},
		{
			// Port of test_simple_move_plan_mixed_files_and_dirs_under_hash_dir
			// (branch 2 wins over branch 3 when any file sits directly in the hash dir).
			name:     "mixed_files_and_dirs_under_hash_dir",
			files:    []string{"torrent_hash/song.mp3", "torrent_hash/album_art/cover.jpg"},
			metadata: map[string]interface{}{"organizer_category": "music"},
			want: map[string]model.PlanAction{
				"torrent_hash": wantMove("music/torrent_hash"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newMockSandbox(t)
			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:      "dl",
				Files:    tc.files,
				Metadata: tc.metadata,
			})
			assertPlanContract(t, code, body, tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 1 (c): legacy archived/app/agents/categorizer/manual_categorizer_test.py
// behavior ports - extension/bango rules drive the exact same outcomes.
// ---------------------------------------------------------------------------

func TestE2E_LegacyPythonManualCategorizerBehavior(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		rules []mock.Rule
		want  map[string]model.PlanAction
	}{
		{
			// Port of test_book_extensions: pure eBook extensions -> book, the
			// whole hash dir is archived (branch 2).
			name: "book_extensions",
			files: []string{
				"ebooks_hash/ebook.pdf",
				"ebooks_hash/novel.epub",
				"ebooks_hash/document.txt",
			},
			want: map[string]model.PlanAction{
				"ebooks_hash": wantMove("book/ebooks_hash"),
			},
		},
		{
			// Port of test_bango_porn_pattern_FC2_case_insensitive: fc2- prefixes
			// are matched case-insensitively and the bango is upper-cased.
			name:  "bango_fc2_case_insensitive",
			files: []string{"fc2-123456.mp4"},
			rules: []mock.Rule{
				{PromptPattern: patBango, Response: `{"filenames":[{"file":"fc2-123456.mp4","new_filename":"FC2-123456.mp4"}]}`},
			},
			want: map[string]model.PlanAction{
				"fc2-123456.mp4": wantMove("jav/素人/FC2-123456.mp4"),
			},
		},
		{
			// Port of test_bango_porn_pattern_3char_3digit: ABC-123.mp4 hits the
			// anchored standard bango rule (Stage 1, no LLM classification).
			name:  "bango_standard_3char_3digit",
			files: []string{"ABC-123.mp4"},
			rules: []mock.Rule{
				{PromptPattern: patBango, Response: `{"filenames":[{"file":"ABC-123.mp4","new_filename":"ABC-123.mp4"}]}`},
			},
			want: map[string]model.PlanAction{
				"ABC-123.mp4": wantMove("jav/素人/ABC-123.mp4"),
			},
		},
		{
			// Port of test_non_video_files_no_highly_possible: a non-video bango
			// name must never be treated as video content; pure eBook rule wins.
			name:  "non_video_bango_names_are_books",
			files: []string{"ABC-123.pdf"},
			want: map[string]model.PlanAction{
				"ABC-123.pdf": wantMove("book/ABC-123.pdf"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, prov := newMockSandbox(t)
			for _, r := range tc.rules {
				prov.AddRule(r)
			}
			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:   "dl",
				Files: tc.files,
			})
			assertPlanContract(t, code, body, tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 1 (d): legacy mover test ports (tv_series_mover_test.py,
// movie_mover_test.py, bango_porn_mover_test.py) driven through the full
// pipeline, with the domain LLM responses mirroring the legacy expected plans.
// ---------------------------------------------------------------------------

func TestE2E_LegacyPythonTVSeriesMover(t *testing.T) {
	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patTVPlanner, Response: `{"plan":[
		{"file":"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv","action":"move","target":"tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.mkv"},
		{"file":"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv","action":"move","target":"tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E02.mkv"}
	]}`})
	prov.AddRule(mock.Rule{PromptPattern: patSubtitle, Response: `{"plan":[
		{"file":"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass","action":"move","matched_video":"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv","language":"English"}
	]}`})

	s.seedDownloadFile(t, "dl", "My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
		"[Script Info]\nTitle: My Date with a Vampire EP1\n")

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir: "dl",
		Files: []string{
			"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv",
			"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv",
			"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass",
			"My.Date.with.a.Vampire.Season.02.2000/cover.jpg",
			"My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part",
		},
		Metadata: map[string]interface{}{"organizer_category": "tv_series", "title": "我和僵尸有个约会", "year": 1998},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.mkv": wantMove("tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.mkv"),
		"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP2.mkv": wantMove("tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E02.mkv"),
		// Legacy expectation: ... S02E01.English.eng.ass placed next to the video.
		"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass": wantMove("tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.English.eng.ass"),
		"My.Date.with.a.Vampire.Season.02.2000/cover.jpg":                                        wantSkip(),
		"My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part":                       wantSkip(),
	})
}

func TestE2E_LegacyPythonMovieMover(t *testing.T) {
	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patMovie, Response: `{"plan":[
		{"file":"The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv","action":"move","target":"movie/Chinese/南海十三郎 (1997)/南海十三郎 (1997).mkv"}
	]}`})
	prov.AddRule(mock.Rule{PromptPattern: patSubtitle, Response: `{"plan":[
		{"file":"The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass","action":"move","matched_video":"The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv","language":"English"}
	]}`})

	s.seedDownloadFile(t, "dl", "The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
		"[Script Info]\nTitle: The Mad Phoenix\n")

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir: "dl",
		Files: []string{
			"The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv",
			"The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass",
			"The.Mad.Phoenix.1997/cover.jpg",
			"The.Mad.Phoenix.1997/behind the scenes.mp4.part",
		},
		Metadata: map[string]interface{}{"organizer_category": "movie", "title": "南海十三郎", "year": 1997},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"The.Mad.Phoenix.1997/The.Mad.Phoenix.1997.mkv":   wantMove("movie/Chinese/南海十三郎 (1997)/南海十三郎 (1997).mkv"),
		"The.Mad.Phoenix.1997/The.Mad.Phoenix.en.ass":     wantMove("movie/Chinese/南海十三郎 (1997)/南海十三郎 (1997).English.eng.ass"),
		"The.Mad.Phoenix.1997/cover.jpg":                  wantSkip(),
		"The.Mad.Phoenix.1997/behind the scenes.mp4.part": wantSkip(),
	})
}

func TestE2E_LegacyPythonBangoMoverWithActor(t *testing.T) {
	s, prov := newMockSandbox(t)

	// Port of bango_porn_mover_test.py: actor "Yua Mikami" is unknown to the
	// local actor file, so the plan creates the actor directory by name.
	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"bango_porn","reason":"standard bango","entities":{"bango":"SSIS-698","actors":["Yua Mikami"]}}`})
	prov.AddRule(mock.Rule{PromptPattern: patBango, Response: `{"filenames":[{"file":"SSIS-698-C.mp4","new_filename":"SSIS-698-C.mp4"}]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "downloads",
		Files: []string{"SSIS-698-C.mp4"},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"SSIS-698-C.mp4": wantMove("jav/Yua Mikami/SSIS-698-C.mp4"),
	})
}

// ---------------------------------------------------------------------------
// Scenario 2: complex release-group noise; main feature renamed, sample skip.
// ---------------------------------------------------------------------------

func TestE2E_ReleaseGroupNoiseMovieWithSample(t *testing.T) {
	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"movie","reason":"release group noise","entities":{"clean_title":"The Matrix","year":1999}}`})
	prov.AddRule(mock.Rule{PromptPattern: patMovie, Response: `{"plan":[
		{"file":"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.mkv","action":"move","target":"movie/English/The Matrix (1999)/The Matrix (1999).mkv"},
		{"file":"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.sample.mkv","action":"skip"}
	]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir: "matrixdl",
		Files: []string{
			"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.mkv",
			"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.sample.mkv",
		},
		Metadata: map[string]interface{}{"title": "The Matrix", "year": 1999},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.mkv":        wantMove("movie/English/The Matrix (1999)/The Matrix (1999).mkv"),
		"[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.sample.mkv": wantSkip(),
	})
}

// ---------------------------------------------------------------------------
// Scenario 3: unconventional episode labels + animation routing (H4):
// [NC-Raws] 葬送的芙莉莲 - 第03话 -> anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/S01E03.
// ---------------------------------------------------------------------------

func TestE2E_AnimeEpisodeRoutingToAnimTVSeries(t *testing.T) {
	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"tv_series","reason":"anime episode","entities":{"clean_title":"葬送的芙莉莲","year":2023}}`})
	prov.AddRule(mock.Rule{PromptPattern: patTVPlanner, Response: `{"plan":[
		{"file":"[NC-Raws] 葬送的芙莉莲 - 第03话/[NC-Raws] 葬送的芙莉莲 - 第03话 [1080p][Baha][WEB-DL].mp4","action":"move","target":"anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/葬送的芙莉莲 (2023) S01E03.mp4"}
	]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir: "frieren",
		Files: []string{
			"[NC-Raws] 葬送的芙莉莲 - 第03话/[NC-Raws] 葬送的芙莉莲 - 第03话 [1080p][Baha][WEB-DL].mp4",
		},
		// "动画" genre flips IsAnim in Stage 2 -> anim_tv_series root (H4).
		Metadata: map[string]interface{}{"genre": "动画"},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"[NC-Raws] 葬送的芙莉莲 - 第03话/[NC-Raws] 葬送的芙莉莲 - 第03话 [1080p][Baha][WEB-DL].mp4": wantMove("anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/葬送的芙莉莲 (2023) S01E03.mp4"),
	})
}

// ---------------------------------------------------------------------------
// Scenario 4: bango derivation matrix and priority (M3):
// ssis-698-a -> part.1, ssis-698-b -> part.2, ssis-698-C keeps its name
// (the -C Chinese-subtitle rule outranks multi-part renumbering).
// ---------------------------------------------------------------------------

func TestE2E_BangoPartMatrixAndCPriority(t *testing.T) {
	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"bango_porn","reason":"bango series","entities":{"bango":"SSIS-698"}}`})
	prov.AddRule(mock.Rule{PromptPattern: patBango, Response: `{"filenames":[
		{"file":"ssis-698-a.mp4","new_filename":"SSIS-698.part.1.mp4"},
		{"file":"ssis-698-b.mp4","new_filename":"SSIS-698.part.2.mp4"},
		{"file":"ssis-698-C.mp4","new_filename":"SSIS-698-C.mp4"}
	]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "javdl",
		Files: []string{"ssis-698-a.mp4", "ssis-698-b.mp4", "ssis-698-C.mp4"},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"ssis-698-a.mp4": wantMove("jav/素人/SSIS-698.part.1.mp4"),
		"ssis-698-b.mp4": wantMove("jav/素人/SSIS-698.part.2.mp4"),
		"ssis-698-C.mp4": wantMove("jav/素人/SSIS-698-C.mp4"),
	})
}

// ---------------------------------------------------------------------------
// Scenario 5: companion subtitle semantic parsing and language alignment:
// Chinese content -> <VideoBaseName>.简体中文.chi.srt, Japanese -> .日本語.jpn.srt.
// ---------------------------------------------------------------------------

func TestE2E_SubtitleSemanticLanguageAlignment(t *testing.T) {
	const matrixVideo = "[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.mkv"
	const matrixSrt = "[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi/The.Matrix.1999.chs.srt"
	const frierenVideo = "[NC-Raws] 葬送的芙莉莲 - 第03话/[NC-Raws] 葬送的芙莉莲 - 第03话 [1080p][Baha][WEB-DL].mp4"
	const frierenSrt = "[NC-Raws] 葬送的芙莉莲 - 第03话/[NC-Raws] 葬送的芙莉莲 - 第03话.jp.srt"

	cases := []struct {
		name string
		dir  string
		// files are the full request input; subFile is seeded on disk so the
		// Stage 4 preview reads real content (L8: DOWNLOAD_COMPLETED_DIR/{dir}/{file}).
		files      []string
		subFile    string
		subContent string
		rules      []mock.Rule
		want       map[string]model.PlanAction
	}{
		{
			name:       "chinese_srt_to_chi_iso_code",
			dir:        "matrixdl",
			files:      []string{matrixVideo, matrixSrt},
			subFile:    matrixSrt,
			subContent: "1\n00:00:01,000 --> 00:00:03,000\n醒醒吧，Neo。\n跟随白兔。\n",
			rules: []mock.Rule{
				{PromptPattern: patClassifier, Response: `{"category":"movie","reason":"feature film","entities":{"clean_title":"The Matrix","year":1999}}`},
				{PromptPattern: patMovie, Response: fmt.Sprintf(`{"plan":[{"file":%q,"action":"move","target":"movie/English/The Matrix (1999)/The Matrix (1999).mkv"}]}`, matrixVideo)},
				{PromptPattern: patSubtitle, Response: fmt.Sprintf(`{"plan":[{"file":%q,"action":"move","matched_video":%q,"language":"Chinese"}]}`, matrixSrt, matrixVideo)},
			},
			want: map[string]model.PlanAction{
				matrixVideo: wantMove("movie/English/The Matrix (1999)/The Matrix (1999).mkv"),
				matrixSrt:   wantMove("movie/English/The Matrix (1999)/The Matrix (1999).简体中文.chi.srt"),
			},
		},
		{
			name:       "japanese_srt_to_jpn_iso_code",
			dir:        "frieren",
			files:      []string{frierenVideo, frierenSrt},
			subFile:    frierenSrt,
			subContent: "1\n00:00:01,000 --> 00:00:03,000\n一度でいいから、\n本物の魔法を見てみたかった。\n",
			rules: []mock.Rule{
				{PromptPattern: patClassifier, Response: `{"category":"tv_series","reason":"anime episode","entities":{"clean_title":"葬送的芙莉莲","year":2023}}`},
				{PromptPattern: patTVPlanner, Response: fmt.Sprintf(`{"plan":[{"file":%q,"action":"move","target":"anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/葬送的芙莉莲 (2023) S01E03.mp4"}]}`, frierenVideo)},
				{PromptPattern: patSubtitle, Response: fmt.Sprintf(`{"plan":[{"file":%q,"action":"move","matched_video":%q,"language":"Japanese"}]}`, frierenSrt, frierenVideo)},
			},
			want: map[string]model.PlanAction{
				frierenVideo: wantMove("anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/葬送的芙莉莲 (2023) S01E03.mp4"),
				frierenSrt:   wantMove("anim_tv_series/Japanese/葬送的芙莉莲 (2023)/Season 01/葬送的芙莉莲 (2023) S01E03.日本語.jpn.srt"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, prov := newMockSandbox(t)
			for _, r := range tc.rules {
				prov.AddRule(r)
			}
			s.seedDownloadFile(t, tc.dir, tc.subFile, tc.subContent)

			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:      tc.dir,
				Files:    tc.files,
				Metadata: map[string]interface{}{"genre": "动画"},
			})
			assertPlanContract(t, code, body, tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 6: multi-file directory-level atomic move (H2) for book /
// audio_book content inside a hash wrapper directory - plan -> execute ->
// filesystem verification -> archive/{dir}.
// ---------------------------------------------------------------------------

func TestE2E_DirectoryLevelAtomicMoveLifecycle(t *testing.T) {
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
			if code != http.StatusOK {
				t.Fatalf("execute expected 200, got %d: %s", code, body)
			}
			var exec model.ExecuteResponse
			decodeBody(t, code, body, &exec)
			if len(exec.FailedMove) != 0 {
				t.Fatalf("expected empty failed_move, got %+v", exec.FailedMove)
			}

			// Every seeded file must have landed under TARGET_DIR.
			for _, f := range tc.files {
				rel := strings.SplitN(f, "/", 2)[1]
				if _, err := os.Stat(filepath.Join(s.targetDir, tc.target, rel)); err != nil {
					t.Fatalf("moved file missing: %v", err)
				}
			}

			// Source directory must be archived to archive/{dir} and be gone.
			if _, err := os.Stat(filepath.Join(s.downloadDir, "archive", tc.dir)); err != nil {
				t.Fatalf("source dir must be archived: %v", err)
			}
			if _, err := os.Stat(filepath.Join(s.downloadDir, tc.dir)); !os.IsNotExist(err) {
				t.Fatalf("source dir must be gone after archive, stat err: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 7: physical execution lifecycle closed loop:
// /v1/plan -> /v1/execute -> file landed in TARGET_DIR -> source dir archived
// to archive/{dir}; plus the /v1/replan-with-hint round trip.
// ---------------------------------------------------------------------------

func TestE2E_FullLifecyclePlanExecuteArchive(t *testing.T) {
	const noiseDir = "[HDSky] The.Matrix.1999.1080p.BluRay.x264.DTS-WiKi"
	video := noiseDir + "/The.Matrix.1999.mkv"
	sample := noiseDir + "/The.Matrix.1999.sample.mkv"

	s, prov := newMockSandbox(t)
	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"movie","reason":"feature film","entities":{"clean_title":"The Matrix","year":1999}}`})
	prov.AddRule(mock.Rule{PromptPattern: patMovie, Response: fmt.Sprintf(`{"plan":[
		{"file":%q,"action":"move","target":"movie/English/The Matrix (1999)/The Matrix (1999).mkv"},
		{"file":%q,"action":"skip"}
	]}`, video, sample)})

	s.seedDownloadFile(t, "matrixdl", video, "mkvdata")
	s.seedDownloadFile(t, "matrixdl", sample, "sampledata")

	// Step 1: plan.
	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:      "matrixdl",
		Files:    []string{video, sample},
		Metadata: map[string]interface{}{"title": "The Matrix", "year": 1999},
	})
	if code != http.StatusOK {
		t.Fatalf("plan expected 200, got %d: %s", code, body)
	}
	var plan model.PlanResponse
	decodeBody(t, code, body, &plan)
	if plan.Error != nil || len(plan.Plan) != 2 {
		t.Fatalf("unexpected plan response: %+v", plan)
	}

	// Step 2: execute the planned actions verbatim.
	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "matrixdl",
		Plan: plan.Plan,
	})
	if code != http.StatusOK {
		t.Fatalf("execute expected 200, got %d: %s", code, body)
	}
	var exec model.ExecuteResponse
	decodeBody(t, code, body, &exec)
	if len(exec.FailedMove) != 0 {
		t.Fatalf("expected empty failed_move, got %+v", exec.FailedMove)
	}

	// Step 3: the main feature must be in TARGET_DIR, the sample dropped.
	if _, err := os.Stat(filepath.Join(s.targetDir, "movie", "English", "The Matrix (1999)", "The Matrix (1999).mkv")); err != nil {
		t.Fatalf("main feature must be in TARGET_DIR: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.targetDir, "movie", "English", "The Matrix (1999)", "The Matrix (1999).sample.mkv")); !os.IsNotExist(err) {
		t.Fatalf("skipped sample must stay out of TARGET_DIR, stat err: %v", err)
	}

	// Step 4: source directory archived to archive/{dir}.
	if _, err := os.Stat(filepath.Join(s.downloadDir, "archive", "matrixdl")); err != nil {
		t.Fatalf("source dir must be archived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.downloadDir, "matrixdl")); !os.IsNotExist(err) {
		t.Fatalf("source dir must be gone after archive, stat err: %v", err)
	}
}

func TestE2E_ReplanWithHintLifecycle(t *testing.T) {
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
	var initial model.PlanResponse
	decodeBody(t, code, body, &initial)
	if code != http.StatusOK || initial.Error != nil || len(initial.Plan) != 1 {
		t.Fatalf("unexpected initial plan (status %d): %s", code, body)
	}

	// Replan with a user hint: stage 1/2 are never re-run.
	code, body = s.postJSON(t, "/v1/replan-with-hint", model.APIReplanRequest{
		Files:            []string{"movie.mkv"},
		Metadata:         map[string]interface{}{"title": "Wrong Name", "year": 2020},
		PreviousResponse: initial,
		UserHint:         "the movie name is wrong, it should be The Correct Name",
	})
	if code != http.StatusOK {
		t.Fatalf("replan expected 200, got %d: %s", code, body)
	}
	var replanned model.PlanResponse
	decodeBody(t, code, body, &replanned)
	if replanned.Error != nil || len(replanned.Plan) != 1 {
		t.Fatalf("unexpected replanned response: %+v", replanned)
	}
	if replanned.Plan[0].Target == nil || *replanned.Plan[0].Target != "movie/English/The Correct Name (2020)/The Correct Name (2020).mkv" {
		t.Fatalf("user hint must be applied, got %+v", replanned.Plan[0])
	}

	// Execute the replanned plan: file lands and source dir is archived.
	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "replandl",
		Plan: replanned.Plan,
	})
	if code != http.StatusOK {
		t.Fatalf("replan execute expected 200, got %d: %s", code, body)
	}
	if _, err := os.Stat(filepath.Join(s.targetDir, "movie", "English", "The Correct Name (2020)", "The Correct Name (2020).mkv")); err != nil {
		t.Fatalf("replanned file must be in TARGET_DIR: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.downloadDir, "archive", "replandl")); err != nil {
		t.Fatalf("source dir must be archived after replan execute: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------

func TestE2E_LiveProviderBusinessLoop(t *testing.T) {
	// Keys and per-provider models come from the gitignored .env.e2e at the
	// repo root; variables already exported in the shell keep precedence.
	// XAI_MODEL and GEMINI_MODEL are independent, so grok and gemini can be
	// exercised in a single run.
	testutil.LoadEnvFile(t, filepath.Join("..", "..", ".env.e2e"))

	targets := []struct {
		modelEnv string
		keyEnv   string
		newProv  func(modelName, apiKey string) ai.Provider
	}{
		{"XAI_MODEL", "XAI_API_KEY", func(m, k string) ai.Provider { return grok.NewProvider(k, ai.WithModel(m)) }},
		{"GEMINI_MODEL", "GEMINI_API_KEY", func(m, k string) ai.Provider { return gemini.NewProvider(k, ai.WithModel(m)) }},
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
			runLiveBusinessLoop(t, target.newProv(modelName, apiKey))
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
	if code != http.StatusOK {
		t.Fatalf("live epub plan expected 200, got %d: %s", code, body)
	}
	var bookPlan model.PlanResponse
	decodeBody(t, code, body, &bookPlan)
	if bookPlan.Error != nil || len(bookPlan.Plan) != 1 ||
		bookPlan.Plan[0].Action != "move" || bookPlan.Plan[0].Target == nil ||
		*bookPlan.Plan[0].Target != "book/MyAwesomeBook.epub" {
		t.Fatalf("unexpected live epub plan: %+v", bookPlan)
	}

	// Real LLM closed loop on a messy movie release: assert structural
	// invariants (never exact LLM wording), then execute and verify delivery.
	const video = "Inception.2010.1080p.BluRay.x264.mkv"
	s.seedDownloadFile(t, "livedl-movie", video, "mkvdata")
	code, body = s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "livedl-movie",
		Files: []string{video},
	})
	if code != http.StatusOK {
		t.Fatalf("live movie plan expected 200, got %d: %s", code, body)
	}
	var moviePlan model.PlanResponse
	decodeBody(t, code, body, &moviePlan)
	if moviePlan.Error != nil {
		t.Fatalf("live movie plan error must be null, got %q", *moviePlan.Error)
	}
	moved := 0
	for _, a := range moviePlan.Plan {
		if a.Action != "move" && a.Action != "skip" {
			t.Fatalf("invalid action %q in live plan %+v", a.Action, a)
		}
		if a.Action == "move" {
			moved++
			if a.Target == nil {
				t.Fatalf("move without target in live plan: %+v", a)
			}
			root := strings.SplitN(*a.Target, "/", 2)[0]
			known := false
			for _, d := range model.AllTargetDirs {
				if string(d) == root {
					known = true
					break
				}
			}
			if !known {
				t.Fatalf("live plan target %q escapes known root dirs", *a.Target)
			}
		}
	}
	if moved != 1 {
		t.Fatalf("exactly one main feature must be moved, got %d moves in %+v", moved, moviePlan.Plan)
	}

	code, body = s.postJSON(t, "/v1/execute", model.APIExecuteRequest{
		Dir:  "livedl-movie",
		Plan: moviePlan.Plan,
	})
	if code != http.StatusOK {
		t.Fatalf("live execute expected 200, got %d: %s", code, body)
	}
	var exec model.ExecuteResponse
	decodeBody(t, code, body, &exec)
	if len(exec.FailedMove) != 0 {
		t.Fatalf("live execute failed_move must be empty, got %+v", exec.FailedMove)
	}
	if _, err := os.Stat(filepath.Join(s.downloadDir, "archive", "livedl-movie")); err != nil {
		t.Fatalf("live source dir must be archived: %v", err)
	}
}
