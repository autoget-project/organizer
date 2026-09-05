package stage1classifier

import (
	"context"
	"encoding/json"
	"fmt"

	"organizer/internal/ai"
	"organizer/internal/model"
)

// ClassifierLLMResponse defines the structured JSON output schema for the Stage 1 Classifier LLM.
type ClassifierLLMResponse struct {
	Category model.Category `json:"category"`
	Reason   string         `json:"reason"`
	Entities struct {
		IMDbID     string   `json:"imdb_id,omitempty"`
		DmmID      string   `json:"dmm_id,omitempty"`
		Bango      string   `json:"bango,omitempty"`
		CleanTitle string   `json:"clean_title,omitempty"`
		Year       int      `json:"year,omitempty"`
		Actors     []string `json:"actors,omitempty"`
	} `json:"entities"`
}

const classifierSystemPrompt = `You are an expert media categorization assistant for a media server organizer.
Analyze the given list of files and optional metadata to classify them into one of the following exact categories:
- "movie": Standalone films/movies, single story movies, animated movies.
- "tv_series": Episodic television shows, series with seasons/episodes (e.g. S01E01, 1x01, EP01, multi-episode batches).
- "photobook": Photography books, photo albums, sets of images/pictures (jpg, png).
- "porn": Western/general adult videos without Japanese bango (JAV) numbering.
- "bango_porn": Japanese adult video (JAV) with bango codes (e.g. SSIS-123, FC2-PPV-123456, Madou series like MD-0123).
- "audio_book": Spoken word audiobooks, chapters of audio books (e.g. "Chapter 1.mp3", "Part 1.mp3", audiobook author/narrator in metadata or tags).
- "book": Text books, novels, documents (pdf, epub, mobi, azw3, txt).
- "music": Musical albums, songs, soundtracks, artist discographies (mp3, flac, wav, ape).
- "music_video": Music videos, live concert recordings, MV.
- "unknown": None of the above or non-media content (e.g. software, games, random noise).

CRITICAL DISAMBIGUATION RULES:
1. Audio files (mp3, flac, wav, etc.):
   - If filenames or metadata indicate chapters, parts, narration, reader, audiobook, novel: classify as "audio_book".
   - If filenames or metadata indicate music albums, tracks, song titles, music artist/album artist: classify as "music".
2. Release Group Noise & Dirty filenames:
   - Strip out release group tags like [HDSky], 1080p.WEB-DL, x264, DDP5.1, Atmos to extract the clean title and year.
3. Mixed formats:
   - If mostly images with a sample video, prefer "photobook".
   - If movie/tv series with subtitle (.srt, .ass) files and nfo, classify based on the main video content.

Extract any identified entities like clean_title, year, imdb_id, dmm_id, or bango if found.
Return your answer strictly matching the required JSON schema.`

// ClassifierLLM handles LLM fallback classification.
type ClassifierLLM struct {
	provider ai.Provider
}

// NewClassifierLLM creates a new ClassifierLLM instance.
func NewClassifierLLM(provider ai.Provider) *ClassifierLLM {
	return &ClassifierLLM{provider: provider}
}

// Classify uses the AI provider to categorize files and extract entities.
func (c *ClassifierLLM) Classify(ctx context.Context, files []string, metadata map[string]interface{}) (model.ClassifierResult, error) {
	if c.provider == nil {
		return model.ClassifierResult{
			Category: model.CategoryUnknown,
			NeedLLM:  true,
		}, fmt.Errorf("classifier llm provider is nil")
	}

	payload := map[string]interface{}{
		"files":    files,
		"metadata": metadata,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return model.ClassifierResult{}, fmt.Errorf("failed to marshal classify input: %w", err)
	}

	prompt := fmt.Sprintf("%s\n\nInput:\n%s", classifierSystemPrompt, string(payloadBytes))

	var resp ClassifierLLMResponse
	if err := c.provider.GenerateStructured(ctx, prompt, ClassifierLLMResponse{}, &resp); err != nil {
		return model.ClassifierResult{}, fmt.Errorf("classifier llm generation failed: %w", err)
	}

	// Validate / normalize category
	cat := resp.Category
	if !isValidCategory(cat) {
		cat = model.CategoryUnknown
	}

	entities := make(map[string]interface{})
	if resp.Entities.IMDbID != "" {
		entities["imdb_id"] = resp.Entities.IMDbID
	}
	if resp.Entities.DmmID != "" {
		entities["dmm_id"] = resp.Entities.DmmID
	}
	if resp.Entities.Bango != "" {
		entities["bango"] = resp.Entities.Bango
	}
	if resp.Entities.CleanTitle != "" {
		entities["clean_title"] = resp.Entities.CleanTitle
	}
	if resp.Entities.Year > 0 {
		entities["year"] = resp.Entities.Year
	}
	if len(resp.Entities.Actors) > 0 {
		entities["actors"] = resp.Entities.Actors
	}
	if resp.Reason != "" {
		entities["reason"] = resp.Reason
	}

	return model.ClassifierResult{
		Category: cat,
		NeedLLM:  true,
		Entities: entities,
	}, nil
}

// ClassifyPipeline executes Rule matcher first, falling back to ClassifierLLM if unmatched.
func ClassifyPipeline(ctx context.Context, provider ai.Provider, files []string, metadata map[string]interface{}) (model.ClassifierResult, error) {
	res, matched := MatchByRules(files, metadata)
	if matched {
		return res, nil
	}

	llm := NewClassifierLLM(provider)
	return llm.Classify(ctx, files, metadata)
}
