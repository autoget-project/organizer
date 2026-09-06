package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

// TestE2E_CategorizerRuleBehavior covers the Stage 1 rule layer over the
// wire: pure eBook extensions archive as a book without any LLM, standard
// bango patterns (case-insensitive FC2) route to the bango mover, and a
// non-video bango name is never treated as video content.
func TestE2E_CategorizerRuleBehavior(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files []string
		rules []mock.Rule
		want  map[string]model.PlanAction
	}{
		{
			// Pure eBook extensions -> book; the whole hash dir is archived
			// (branch 2 of the simple mover).
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
			// FC2- prefixes match case-insensitively and the bango is
			// upper-cased.
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
			// ABC-123.mp4 hits the anchored standard bango rule (Stage 1, no
			// LLM classification).
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
			// A non-video bango name must never be treated as video content;
			// the pure eBook rule wins.
			name:  "non_video_bango_names_are_books",
			files: []string{"ABC-123.pdf"},
			want: map[string]model.PlanAction{
				"ABC-123.pdf": wantMove("book/ABC-123.pdf"),
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

			code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
				Dir:   "dl",
				Files: tc.files,
			})
			assertPlanContract(t, code, body, tc.want)

			// The pure eBook rules must hold with zero LLM calls.
			if len(tc.rules) == 0 {
				require.Empty(t, prov.Calls(), "rule-only classification must not invoke the LLM")
			}
		})
	}
}
