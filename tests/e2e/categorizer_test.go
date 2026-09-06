package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/model"
)

// TestE2E_CategorizerRuleBehavior covers Stage 1 rule classification:
// pure eBook extensions archive as book, standard bango patterns route to bango mover,
// and pure eBook bango name is treated as book.
func TestE2E_CategorizerRuleBehavior(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  map[string]model.PlanAction
	}{
		{
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
			name:  "bango_fc2_case_insensitive",
			files: []string{"fc2-123456.mp4"},
			want: map[string]model.PlanAction{
				"fc2-123456.mp4": wantMove("jav/素人/FC2-123456.mp4"),
			},
		},
		{
			name:  "bango_standard_3char_3digit",
			files: []string{"ABC-123.mp4"},
			want: map[string]model.PlanAction{
				"ABC-123.mp4": wantMove("jav/素人/ABC-123.mp4"),
			},
		},
		{
			name:  "non_video_bango_names_are_books",
			files: []string{"ABC-123.pdf"},
			want: map[string]model.PlanAction{
				"ABC-123.pdf": wantMove("book/ABC-123.pdf"),
			},
		},
	}

	runWithLiveProviders(t, func(t *testing.T, s *sandbox) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
					Dir:   "dl",
					Files: tc.files,
				})
				assertPlanContract(t, code, body, tc.want)
			})
		}
	})
}

// TestE2E_CategorizerLLMDisambiguation tests dirty/ambiguous inputs that require LLM classification:
// audiobooks (chapter mp3s) vs western porn (no bango code).
func TestE2E_CategorizerLLMDisambiguation(t *testing.T) {
	runWithLiveProviders(t, func(t *testing.T, s *sandbox) {
		t.Run("audiobook_disambiguation", func(t *testing.T) {
			files := []string{
				"The_Hobbit/Chapter_01.mp3",
				"The_Hobbit/Chapter_02.mp3",
			}
			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:   "audiobookdl",
				Files: files,
			})
			require.Equal(t, 200, code, body)
			var resp model.PlanResponse
			decodeBody(t, code, body, &resp)
			assert.Nil(t, resp.Error)
			require.NotEmpty(t, resp.Plan)

			// Invariant: should plan under audio_book
			for _, act := range resp.Plan {
				assert.Equal(t, "move", act.Action)
				require.NotNil(t, act.Target)
				assert.Contains(t, *act.Target, "audio_book/")
			}
		})

		t.Run("western_porn_disambiguation", func(t *testing.T) {
			files := []string{
				"Brazzers.Exxtra.Hot.Summer.Scenes.1080p.mp4",
			}
			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:   "porndl",
				Files: files,
			})
			require.Equal(t, 200, code, body)
			var resp model.PlanResponse
			decodeBody(t, code, body, &resp)
			assert.Nil(t, resp.Error)
			require.NotEmpty(t, resp.Plan)
			assert.Equal(t, "move", resp.Plan[0].Action)
			require.NotNil(t, resp.Plan[0].Target)
			// Invariant: should plan under porn/
			assert.Contains(t, *resp.Plan[0].Target, "porn/")
		})
	})
}
