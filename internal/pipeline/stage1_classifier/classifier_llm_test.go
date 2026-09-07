package stage1classifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/model"
)

func TestClassifierLLM_AudioDisambiguation(t *testing.T) {
	t.Parallel()

	mockProv := mock.NewProvider()
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Audiobook - Chapter 1.mp3",
		Response: ClassifierLLMResponse{
			Category: model.CategoryAudioBook,
			Reason:   "Filenames represent chapters of an audiobook",
		},
	})
	mockProv.AddRule(mock.Rule{
		PromptPattern: "01. Taylor Swift - Blank Space.flac",
		Response: ClassifierLLMResponse{
			Category: model.CategoryMusic,
			Reason:   "Track with music artist and song title",
		},
	})

	llm := NewClassifierLLM(mockProv)

	tests := []struct {
		name         string
		files        []string
		wantCategory model.Category
	}{
		{"audio book chapters", []string{"Audiobook - Chapter 1.mp3", "Audiobook - Chapter 2.mp3"}, model.CategoryAudioBook},
		{"music track", []string{"01. Taylor Swift - Blank Space.flac"}, model.CategoryMusic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := llm.Classify(context.Background(), tt.files, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCategory, res.Category)
		})
	}
}

func TestClassifyPipeline_Integration(t *testing.T) {
	t.Parallel()

	mockProv := mock.NewProvider()
	mockProv.SetDefaultResponse(ClassifierLLMResponse{
		Category: model.CategoryMovie,
		Reason:   "Extracted movie from noisy release group string",
	}, nil)
	ctx := context.Background()

	// Matched by rule (pure book): the LLM must never be called.
	res1, err := ClassifyPipeline(ctx, mockProv, []string{"document.pdf"}, nil)
	require.NoError(t, err)
	assert.False(t, res1.NeedLLM)
	assert.Equal(t, model.CategoryBook, res1.Category)
	assert.Empty(t, mockProv.Calls(), "rule match must not invoke the LLM")

	// Unmatched by rule (dirty filename): the LLM must be invoked.
	res2, err := ClassifyPipeline(ctx, mockProv, []string{"[HDSky] Inception.2010.1080p.mkv"}, nil)
	require.NoError(t, err)
	assert.True(t, res2.NeedLLM)
	assert.Equal(t, model.CategoryMovie, res2.Category)
	assert.NotEmpty(t, mockProv.Calls(), "LLM must be called for dirty filename")
}

func TestClassifierLLM_GirlsWayPornVsJAV(t *testing.T) {
	t.Parallel()

	mockProv := mock.NewProvider()
	// Porn specialist returns "yes"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Western/general adult video (porn)",
		Response: CheckerResponse{
			Confidence: ConfidenceYes,
			Reason:     "Western adult release with performer names Khloe Kapri and Megan Mistakes",
			Entities: CheckerEntities{
				CleanTitle: "Megan Mistakes Runaway Brides Regret",
				Year:       2026,
				Actors:     []string{"Khloe Kapri", "Megan Mistakes"},
			},
		},
	})
	// JAV bango specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Japanese adult video (JAV)",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "No Japanese bango code found",
		},
	})
	// Movie specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "standalone movie or film",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "Adult video release, not mainstream movie",
		},
	})
	// TV specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "episodic TV series",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "Not an episodic TV series",
		},
	})
	// Music video specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "music video (MV)",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "Not a music video",
		},
	})

	llm := NewClassifierLLM(mockProv)
	files := []string{"GirlsWay Khloe Kapri Megan Mistakes Runaway Brides Regret 2026 2160p WEB-DL H264 AAC2.0-VSEX.mp4"}

	res, err := llm.Classify(context.Background(), files, nil)
	require.NoError(t, err)
	assert.Equal(t, model.CategoryPorn, res.Category, "GirlsWay release must be classified as porn, not bango_porn")
	assert.Equal(t, "Megan Mistakes Runaway Brides Regret", res.Entities["clean_title"])
	assert.Equal(t, 2026, res.Entities["year"])
}

func TestClassifierLLM_ArbiterResolvesConflict(t *testing.T) {
	t.Parallel()

	mockProv := mock.NewProvider()
	// Both porn and movie return "maybe"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Western/general adult video (porn)",
		Response: CheckerResponse{
			Confidence: ConfidenceMaybe,
			Reason:     "Might contain adult themes",
		},
	})
	mockProv.AddRule(mock.Rule{
		PromptPattern: "standalone movie or film",
		Response: CheckerResponse{
			Confidence: ConfidenceMaybe,
			Reason:     "Looks like a feature film",
		},
	})
	// TV specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "episodic TV series",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "Not episodic",
		},
	})
	// JAV bango specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Japanese adult video (JAV)",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "No JAV bango",
		},
	})
	// Music video specialist returns "no"
	mockProv.AddRule(mock.Rule{
		PromptPattern: "music video (MV)",
		Response: CheckerResponse{
			Confidence: ConfidenceNo,
			Reason:     "Not a music video",
		},
	})
	// Arbiter rules in favor of movie
	mockProv.AddRule(mock.Rule{
		PromptPattern: "categorization arbiter",
		Response: ArbiterDecision{
			Category: model.CategoryMovie,
			Reason:   "Arbiter decided it is an indie art movie rather than adult video",
			Entities: CheckerEntities{
				CleanTitle: "Art Movie",
				Year:       2022,
			},
		},
	})

	llm := NewClassifierLLM(mockProv)
	files := []string{"Art Movie 2022.mkv"}

	res, err := llm.Classify(context.Background(), files, nil)
	require.NoError(t, err)
	assert.Equal(t, model.CategoryMovie, res.Category)
	assert.Equal(t, "Art Movie", res.Entities["clean_title"])
}
