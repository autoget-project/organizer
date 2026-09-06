package e2e

import (
	"testing"

	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/model"
)

// TestE2E_TVSeriesMover drives the TV series mover through the full pipeline:
// episodes land in the Season directory, the companion subtitle is renamed
// next to its matched video, and cover art / partial files are skipped.
func TestE2E_TVSeriesMover(t *testing.T) {
	t.Parallel()

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
		// The subtitle lands next to its matched video as S02E01.English.eng.ass.
		"My.Date.with.a.Vampire.Season.02.2000/My.Date.with.a.Vampire.Season.02.2000.EP1.en.ass": wantMove("tv_series/Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.English.eng.ass"),
		"My.Date.with.a.Vampire.Season.02.2000/cover.jpg":                                        wantSkip(),
		"My.Date.with.a.Vampire.Season.02.2000/behind the scenes.mp4.part":                       wantSkip(),
	})
}

// TestE2E_AnimeEpisodeRoutingToAnimTVSeries covers unconventional episode
// labels plus animation routing (H4): a CJK-numbered anime episode with the
// "动画" genre flag routes to anim_tv_series as Season 01 / S01E03.
func TestE2E_AnimeEpisodeRoutingToAnimTVSeries(t *testing.T) {
	t.Parallel()

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
