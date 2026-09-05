package stage1classifier

import (
	"context"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

func TestClassifierLLM_AudioDisambiguation(t *testing.T) {
	mockProv := mock.NewProvider()

	// Rule 1: Audiobook chapters
	mockProv.AddRule(mock.Rule{
		PromptPattern: "Audiobook - Chapter 1.mp3",
		Response: ClassifierLLMResponse{
			Category: model.CategoryAudioBook,
			Reason:   "Filenames represent chapters of an audiobook",
		},
	})

	// Rule 2: Music album
	mockProv.AddRule(mock.Rule{
		PromptPattern: "01. Taylor Swift - Blank Space.flac",
		Response: ClassifierLLMResponse{
			Category: model.CategoryMusic,
			Reason:   "Track with music artist and song title",
		},
	})

	llm := NewClassifierLLM(mockProv)
	ctx := context.Background()

	// Test 1: Audio book sample
	res1, err := llm.Classify(ctx, []string{"Audiobook - Chapter 1.mp3", "Audiobook - Chapter 2.mp3"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.Category != model.CategoryAudioBook {
		t.Fatalf("expected CategoryAudioBook, got %s", res1.Category)
	}

	// Test 2: Music sample
	res2, err := llm.Classify(ctx, []string{"01. Taylor Swift - Blank Space.flac"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Category != model.CategoryMusic {
		t.Fatalf("expected CategoryMusic, got %s", res2.Category)
	}
}

func TestClassifyPipeline_Integration(t *testing.T) {
	mockProv := mock.NewProvider()
	mockProv.SetDefaultResponse(ClassifierLLMResponse{
		Category: model.CategoryMovie,
		Reason:   "Extracted movie from noisy release group string",
	}, nil)

	ctx := context.Background()

	// 1. Matched by rule (pure book) -> should NOT call LLM
	res1, err := ClassifyPipeline(ctx, mockProv, []string{"document.pdf"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.Category != model.CategoryBook || res1.NeedLLM {
		t.Fatalf("expected CategoryBook without LLM, got %s (%t)", res1.Category, res1.NeedLLM)
	}
	if len(mockProv.Calls()) != 0 {
		t.Fatalf("expected 0 mock calls, got %d", len(mockProv.Calls()))
	}

	// 2. Unmatched by rule (dirty filename) -> should invoke LLM
	res2, err := ClassifyPipeline(ctx, mockProv, []string{"[HDSky] Inception.2010.1080p.mkv"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Category != model.CategoryMovie || !res2.NeedLLM {
		t.Fatalf("expected CategoryMovie with LLM, got %s (%t)", res2.Category, res2.NeedLLM)
	}
	if len(mockProv.Calls()) != 1 {
		t.Fatalf("expected 1 mock call, got %d", len(mockProv.Calls()))
	}
}
