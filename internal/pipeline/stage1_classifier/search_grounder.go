package stage1classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/autoget-project/organizer/internal/ai"
)

// SearchContext contains factual background discovered via search.
type SearchContext struct {
	SearchSummary string   `json:"search_summary,omitempty"`
	DetectedType  string   `json:"detected_type,omitempty"`
	OfficialTitle string   `json:"official_title,omitempty"`
	Studio        string   `json:"studio,omitempty"`
	Actors        []string `json:"actors,omitempty"`
	Year          int      `json:"year,omitempty"`
}

// HasInfo checks if any factual information was found.
func (s SearchContext) HasInfo() bool {
	return s.SearchSummary != "" || s.DetectedType != "" || s.OfficialTitle != "" || s.Studio != "" || len(s.Actors) > 0
}

const searchGrounderPrompt = `You are a media intelligence search assistant.
Your task is to search Google to identify what this downloaded media release actually is.

Instructions:
1. Strip release noise (e.g. resolution 2160p, 1080p, encoding WEB-DL, H264, AAC2.0, release groups -VSEX, -HDChina).
2. Search Google for the core title, series name, studio, or performer names.
3. Extract verified real-world facts:
   - search_summary: 1-2 sentence factual summary of what work/release this is.
   - detected_type: identified type if found (e.g. "Western adult video / porn", "Japanese adult video / JAV", "movie", "tv_series", "music", "audiobook", or "unknown").
   - official_title: clean official title.
   - studio: production studio, brand, or network (e.g. "GirlsWay", "Brazzers", "Netflix", "HBO", "S1").
   - actors: performer names recognized from the release.
   - year: release year if confirmed.

Return your answer strictly matching the required JSON schema.`

// GroundWithSearch runs a single search query across the files if the provider supports SearchProvider.
// If provider does not support search or search fails, it returns an empty SearchContext gracefully without failing.
func GroundWithSearch(ctx context.Context, provider ai.Provider, files []string, metadata map[string]interface{}) SearchContext {
	sp, ok := provider.(ai.SearchProvider)
	if !ok {
		log.Printf("stage1 search grounding: provider %s does not support search, skipped", provider.Name())
		return SearchContext{}
	}

	payload := map[string]interface{}{
		"files":    files,
		"metadata": metadata,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return SearchContext{}
	}

	prompt := fmt.Sprintf("%s\n\nInput:\n%s", searchGrounderPrompt, string(payloadBytes))
	var result SearchContext
	if err := sp.GenerateStructuredWithSearch(ctx, prompt, SearchContext{}, &result); err != nil {
		// Non-fatal: degrade gracefully if search fails
		log.Printf("stage1 search grounding failed, continuing without search context: %v", err)
		return SearchContext{}
	}
	log.Printf("stage1 search grounding: detected_type=%q official_title=%q studio=%q year=%d actors=%v summary=%q",
		result.DetectedType, result.OfficialTitle, result.Studio, result.Year, result.Actors, result.SearchSummary)
	return result
}
