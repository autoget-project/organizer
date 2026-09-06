package e2e

import (
	"testing"

	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/model"
)

// TestE2E_MovieMover drives the movie mover through the full pipeline: the
// main feature is renamed into the movie library, the companion subtitle is
// renamed next to it, and cover art / partial files are skipped.
func TestE2E_MovieMover(t *testing.T) {
	t.Parallel()

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

// TestE2E_ReleaseGroupNoiseMovieWithSample covers a complex release with
// release-group noise: the main feature is renamed into the library while the
// sample file is skipped.
func TestE2E_ReleaseGroupNoiseMovieWithSample(t *testing.T) {
	t.Parallel()

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
