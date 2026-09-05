package stage2enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"organizer/internal/model"
)

type mockMCPClient struct {
	findByIMDbIDFunc       func(ctx context.Context, imdbID string) (map[string]interface{}, error)
	searchMoviesFunc       func(ctx context.Context, title string) (map[string]interface{}, error)
	searchTVShowsFunc      func(ctx context.Context, title string) (map[string]interface{}, error)
	searchJapanesePornFunc func(ctx context.Context, javID string) (map[string]interface{}, error)
}

func (m *mockMCPClient) CallTool(ctx context.Context, name string, arguments interface{}) (json.RawMessage, error) {
	return nil, nil
}
func (m *mockMCPClient) FindByIMDbID(ctx context.Context, imdbID string) (map[string]interface{}, error) {
	if m.findByIMDbIDFunc != nil {
		return m.findByIMDbIDFunc(ctx, imdbID)
	}
	return nil, nil
}
func (m *mockMCPClient) SearchMovies(ctx context.Context, title string) (map[string]interface{}, error) {
	if m.searchMoviesFunc != nil {
		return m.searchMoviesFunc(ctx, title)
	}
	return nil, nil
}
func (m *mockMCPClient) SearchTVShows(ctx context.Context, title string) (map[string]interface{}, error) {
	if m.searchTVShowsFunc != nil {
		return m.searchTVShowsFunc(ctx, title)
	}
	return nil, nil
}
func (m *mockMCPClient) SearchJapanesePorn(ctx context.Context, javID string) (map[string]interface{}, error) {
	if m.searchJapanesePornFunc != nil {
		return m.searchJapanesePornFunc(ctx, javID)
	}
	return nil, nil
}
func (m *mockMCPClient) WebSearch(ctx context.Context, query string) (map[string]interface{}, error) {
	return nil, nil
}

func TestEnricher_MovieSuccessAndAnimation(t *testing.T) {
	mcpMock := &mockMCPClient{
		findByIMDbIDFunc: func(ctx context.Context, imdbID string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"movie_results": []interface{}{
					map[string]interface{}{
						"title":             "Spirited Away",
						"release_date":      "2001-07-20",
						"original_language": "ja",
						"genres": []interface{}{
							map[string]interface{}{"name": "Animation"},
						},
					},
				},
			}, nil
		},
	}

	enricher := NewEnricher(mcpMock, nil, nil)
	ctx := context.Background()

	meta := map[string]interface{}{"imdb_id": "tt0245429"}
	res, err := enricher.Enrich(ctx, model.CategoryMovie, []string{"Spirited.Away.2001.mkv"}, meta, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Title != "Spirited Away" {
		t.Fatalf("expected Spirited Away, got %s", res.Title)
	}
	if res.Year != 2001 {
		t.Fatalf("expected year 2001, got %d", res.Year)
	}
	if !res.IsAnim {
		t.Fatalf("expected IsAnim to be true")
	}
	if res.Language != model.LanguageJapanese {
		t.Fatalf("expected LanguageJapanese, got %s", res.Language)
	}
}

func TestEnricher_DegradationProtection_M6(t *testing.T) {
	// M6: If MCP throws error or returns empty, enricher MUST NOT return fatal 500 error!
	mcpMock := &mockMCPClient{
		findByIMDbIDFunc: func(ctx context.Context, imdbID string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("network connection refused")
		},
		searchMoviesFunc: func(ctx context.Context, title string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("search error")
		},
		searchJapanesePornFunc: func(ctx context.Context, javID string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("remote timeout")
		},
	}

	tmpDir := t.TempDir()
	store := NewActorStore(filepath.Join(tmpDir, "actor.json"), "", nil, nil)
	enricher := NewEnricher(mcpMock, store, nil)
	ctx := context.Background()

	// 1. Movie fallback to filename
	resMovie, err := enricher.Enrich(ctx, model.CategoryMovie, []string{"Inception.2010.mkv"}, map[string]interface{}{"imdb_id": "tt9999999"}, nil)
	if err != nil {
		t.Fatalf("expected NO error on movie degradation, got %v", err)
	}
	if resMovie.Title == "" || resMovie.Year != 2010 {
		t.Fatalf("expected degraded movie metadata from filename, got Title=%s Year=%d", resMovie.Title, resMovie.Year)
	}

	// 2. Bango fallback to filename regex
	resBango, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"SSIS-001.mp4"}, nil, nil)
	if err != nil {
		t.Fatalf("expected NO error on bango degradation, got %v", err)
	}
	if resBango.Bango != "SSIS-001" {
		t.Fatalf("expected Bango SSIS-001, got %s", resBango.Bango)
	}
}

func TestEnricher_BangoVRAndMadou(t *testing.T) {
	enricher := NewEnricher(nil, nil, nil)
	ctx := context.Background()

	// Madou test
	resMadou, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"MDCM-0005.mp4"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resMadou.FromMadou || resMadou.Language != model.LanguageChinese {
		t.Fatalf("expected FromMadou=true and LanguageChinese, got %t %s", resMadou.FromMadou, resMadou.Language)
	}

	// Madou exact-label positives (full label set must be recognized)
	for _, bango := range []string{"MD-003", "MDSR-001", "MDHG-012", "MDHT-004", "MDL-002", "MSD-0001"} {
		res, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{bango + ".mp4"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", bango, err)
		}
		if !res.FromMadou || res.Language != model.LanguageChinese {
			t.Fatalf("expected %s to be Madou (FromMadou=true, LanguageChinese), got %t %s", bango, res.FromMadou, res.Language)
		}
	}

	// Madou misclassification negatives: mainstream JAV labels starting with
	// "MD" must NOT be flagged as Madou.
	for _, bango := range []string{"MIDE-612", "MIDD-826", "MDBK-018", "MDYD-528", "MDX-001"} {
		res, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{bango + ".mp4"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", bango, err)
		}
		if res.FromMadou {
			t.Fatalf("expected %s NOT to be flagged as Madou", bango)
		}
		if res.Language != model.LanguageJapanese {
			t.Fatalf("expected %s to keep LanguageJapanese, got %s", bango, res.Language)
		}
	}

	// VR test
	resVR, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"IPVR-002.mp4"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resVR.IsVR {
		t.Fatalf("expected IsVR=true")
	}
}

func TestEnricher_BangoDmmKeyFallsBackToFilenameBango(t *testing.T) {
	// Spec M6: the dmm_id is only a search key; when search_japanese_porn
	// fails or returns nothing, the final bango must fall back to the
	// filename-derived canonical hyphenated bango.
	tests := []struct {
		name     string
		mockFunc func(ctx context.Context, javID string) (map[string]interface{}, error)
	}{
		{
			name: "mcp error",
			mockFunc: func(ctx context.Context, javID string) (map[string]interface{}, error) {
				if javID != "PRED00374" {
					t.Errorf("expected dmm-derived search key PRED00374, got %s", javID)
				}
				return nil, fmt.Errorf("remote timeout")
			},
		},
		{
			name: "empty result",
			mockFunc: func(ctx context.Context, javID string) (map[string]interface{}, error) {
				if javID != "PRED00374" {
					t.Errorf("expected dmm-derived search key PRED00374, got %s", javID)
				}
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpMock := &mockMCPClient{searchJapanesePornFunc: tt.mockFunc}
			enricher := NewEnricher(mcpMock, nil, nil)

			res, err := enricher.Enrich(
				context.Background(),
				model.CategoryBangoPorn,
				[]string{"PRED-374.mp4"},
				map[string]interface{}{"dmm_id": "pred00374"},
				nil,
			)
			if err != nil {
				t.Fatalf("expected NO error on bango degradation, got %v", err)
			}
			if res.Bango != "PRED-374" {
				t.Fatalf("expected fallback Bango PRED-374, got %s", res.Bango)
			}
		})
	}
}
