package e2e

import (
	"fmt"
	"testing"

	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/model"
)

// TestE2E_SubtitleMoverSemanticLanguage covers companion subtitle semantic
// pairing and language alignment (Stage 4): Chinese content renames the
// subtitle to <VideoBaseName>.简体中文.chi.srt, Japanese content to
// <VideoBaseName>.日本語.jpn.srt, each placed next to its matched video.
func TestE2E_SubtitleMoverSemanticLanguage(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
