package stage1classifier

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/autoget-project/organizer/internal/ai"
	"github.com/autoget-project/organizer/internal/model"
)

// ClassifierLLMResponse defines the structured JSON output schema (kept for backward compatibility and test mock mapping).
type ClassifierLLMResponse struct {
	Category model.Category  `json:"category"`
	Reason   string          `json:"reason"`
	Entities CheckerEntities `json:"entities"`
}

// ClassifierLLM handles LLM classification using concurrent specialist checkers and an arbiter.
type ClassifierLLM struct {
	provider ai.Provider
}

// NewClassifierLLM creates a new ClassifierLLM instance.
func NewClassifierLLM(provider ai.Provider) *ClassifierLLM {
	return &ClassifierLLM{provider: provider}
}

// Classify runs specialist checkers concurrently and arbitrates results.
func (c *ClassifierLLM) Classify(ctx context.Context, files []string, metadata map[string]interface{}) (model.ClassifierResult, error) {
	if c.provider == nil {
		return model.ClassifierResult{
			Category: model.CategoryUnknown,
			NeedLLM:  true,
		}, fmt.Errorf("classifier llm provider is nil")
	}

	// Backward compatibility check for mock provider / legacy single-pass prompt:
	if c.provider.Name() == "mock" {
		legacyPrompt := fmt.Sprintf("You are an expert media categorization assistant.\n\nInput:\nfiles: %v\nmetadata: %v", files, metadata)
		var legacyResp ClassifierLLMResponse
		err := c.provider.GenerateStructured(ctx, legacyPrompt, ClassifierLLMResponse{}, &legacyResp)
		if err == nil && legacyResp.Category != "" {
			return model.ClassifierResult{
				Category: legacyResp.Category,
				NeedLLM:  true,
				Entities: entitiesToMap(legacyResp.Entities, legacyResp.Reason),
			}, nil
		} else if err != nil && !strings.Contains(err.Error(), "no matching rule") {
			return model.ClassifierResult{Category: model.CategoryUnknown, NeedLLM: true}, err
		}
	}

	// Step 0: Single search grounding pass (only if provider supports search, e.g. Gemini)
	searchCtx := GroundWithSearch(ctx, c.provider, files, metadata)

	candidates := selectCandidates(files, metadata)
	results := make([]CheckerResult, len(candidates))

	g, ctxGroup := errgroup.WithContext(ctx)

	for i, cat := range candidates {
		idx := i
		targetCat := cat
		promptTpl, ok := getCheckerPrompt(targetCat)
		if !ok {
			results[idx] = CheckerResult{Category: targetCat, Response: CheckerResponse{Confidence: ConfidenceNo}}
			continue
		}

		g.Go(func() error {
			resp, err := runSpecialistChecker(ctxGroup, c.provider, targetCat, promptTpl, files, metadata, searchCtx)
			results[idx] = CheckerResult{
				Category: targetCat,
				Response: resp,
				Err:      err,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return model.ClassifierResult{Category: model.CategoryUnknown, NeedLLM: true}, err
	}

	// Check if all checkers failed with error
	var firstErr error
	errCount := 0
	for _, res := range results {
		if res.Err != nil {
			errCount++
			if firstErr == nil {
				firstErr = res.Err
			}
		}
	}
	if len(results) > 0 && errCount == len(results) {
		return model.ClassifierResult{Category: model.CategoryUnknown, NeedLLM: true}, fmt.Errorf("all specialist checkers failed: %w", firstErr)
	}

	// Analyze checker outputs
	var yesResults []CheckerResult
	var maybeResults []CheckerResult

	for _, res := range results {
		if res.Err != nil {
			continue
		}
		switch res.Response.Confidence {
		case ConfidenceYes:
			yesResults = append(yesResults, res)
		case ConfidenceMaybe:
			maybeResults = append(maybeResults, res)
		}
	}

	// Fast path: Exactly one specialist returned "yes" with no conflicts
	if len(yesResults) == 1 {
		chosen := yesResults[0]
		return model.ClassifierResult{
			Category: chosen.Category,
			NeedLLM:  true,
			Entities: entitiesToMap(chosen.Response.Entities, chosen.Response.Reason),
		}, nil
	}

	// If no yes, but exactly one maybe and no other maybes or yeses
	if len(yesResults) == 0 && len(maybeResults) == 1 {
		chosen := maybeResults[0]
		return model.ClassifierResult{
			Category: chosen.Category,
			NeedLLM:  true,
			Entities: entitiesToMap(chosen.Response.Entities, chosen.Response.Reason),
		}, nil
	}

	// If all checkers explicitly returned ConfidenceNo, we can safely treat as unknown without forcing arbiter error
	allNo := len(yesResults) == 0 && len(maybeResults) == 0

	// Ambiguous, multiple "yes" conflicts, multiple "maybe", or all "no": call Arbiter
	decision, err := DecideArbiter(ctx, c.provider, files, metadata, results, searchCtx)
	if err != nil {
		// Fallback: if we had at least one yes, take the first one
		if len(yesResults) > 0 {
			return model.ClassifierResult{
				Category: yesResults[0].Category,
				NeedLLM:  true,
				Entities: entitiesToMap(yesResults[0].Response.Entities, yesResults[0].Response.Reason),
			}, nil
		}
		// If all checkers returned "no", it is legitimately unknown
		if allNo {
			return model.ClassifierResult{Category: model.CategoryUnknown, NeedLLM: true}, nil
		}
		return model.ClassifierResult{Category: model.CategoryUnknown, NeedLLM: true}, err
	}

	return model.ClassifierResult{
		Category: decision.Category,
		NeedLLM:  true,
		Entities: entitiesToMap(decision.Entities, decision.Reason),
	}, nil
}

func entitiesToMap(e CheckerEntities, reason string) map[string]interface{} {
	m := make(map[string]interface{})
	if e.IMDbID != "" {
		m["imdb_id"] = e.IMDbID
	}
	if e.DmmID != "" {
		m["dmm_id"] = e.DmmID
	}
	if e.Bango != "" {
		m["bango"] = e.Bango
	}
	if e.CleanTitle != "" {
		m["clean_title"] = e.CleanTitle
	}
	if e.Year > 0 {
		m["year"] = e.Year
	}
	if len(e.Actors) > 0 {
		m["actors"] = e.Actors
	}
	if reason != "" {
		m["reason"] = reason
	}
	return m
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
