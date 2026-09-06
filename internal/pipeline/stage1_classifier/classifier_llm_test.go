package stage1classifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
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
	assert.Len(t, mockProv.Calls(), 1)
}
