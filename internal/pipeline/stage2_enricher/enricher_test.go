package stage2enricher

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	t.Parallel()

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

	res, err := enricher.Enrich(context.Background(), model.CategoryMovie,
		[]string{"Spirited.Away.2001.mkv"}, map[string]interface{}{"imdb_id": "tt0245429"}, nil)
	require.NoError(t, err)

	assert.Equal(t, "Spirited Away", res.Title)
	assert.Equal(t, 2001, res.Year)
	assert.True(t, res.IsAnim)
	assert.Equal(t, model.LanguageJapanese, res.Language)
}

func TestEnricher_DegradationProtection_M6(t *testing.T) {
	t.Parallel()

	// M6: when MCP fails or returns nothing, the enricher must degrade to
	// filename-derived metadata instead of surfacing a fatal error.
	mcpMock := &mockMCPClient{
		findByIMDbIDFunc: func(ctx context.Context, imdbID string) (map[string]interface{}, error) {
			return nil, errors.New("network connection refused")
		},
		searchMoviesFunc: func(ctx context.Context, title string) (map[string]interface{}, error) {
			return nil, errors.New("search error")
		},
		searchJapanesePornFunc: func(ctx context.Context, javID string) (map[string]interface{}, error) {
			return nil, errors.New("remote timeout")
		},
	}

	store := NewActorStore(filepath.Join(t.TempDir(), "actor.json"), "", nil)
	enricher := NewEnricher(mcpMock, store, nil)
	ctx := context.Background()

	// Movie falls back to the filename.
	resMovie, err := enricher.Enrich(ctx, model.CategoryMovie,
		[]string{"Inception.2010.mkv"}, map[string]interface{}{"imdb_id": "tt9999999"}, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, resMovie.Title)
	assert.Equal(t, 2010, resMovie.Year)

	// Bango falls back to the filename-derived bango.
	resBango, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"SSIS-001.mp4"}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "SSIS-001", resBango.Bango)
}

func TestEnricher_BangoVRAndMadou(t *testing.T) {
	t.Parallel()

	enricher := NewEnricher(nil, nil, nil)
	ctx := context.Background()

	// Madou label detection.
	resMadou, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"MDCM-0005.mp4"}, nil, nil)
	require.NoError(t, err)
	assert.True(t, resMadou.FromMadou)
	assert.Equal(t, model.LanguageChinese, resMadou.Language)

	// Every exact madou label prefix must be recognized.
	for _, bango := range []string{"MD-003", "MDSR-001", "MDHG-012", "MDHT-004", "MDL-002", "MSD-0001"} {
		res, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{bango + ".mp4"}, nil, nil)
		require.NoError(t, err)
		assert.True(t, res.FromMadou, "expected %s to be Madou", bango)
		assert.Equal(t, model.LanguageChinese, res.Language, "expected %s to be Chinese", bango)
	}

	// Mainstream JAV labels starting with "MD" must NOT be flagged as Madou.
	for _, bango := range []string{"MIDE-612", "MIDD-826", "MDBK-018", "MDYD-528", "MDX-001"} {
		res, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{bango + ".mp4"}, nil, nil)
		require.NoError(t, err)
		assert.False(t, res.FromMadou, "expected %s NOT to be Madou", bango)
		assert.Equal(t, model.LanguageJapanese, res.Language, "expected %s to keep Japanese", bango)
	}

	// VR detection.
	resVR, err := enricher.Enrich(ctx, model.CategoryBangoPorn, []string{"IPVR-002.mp4"}, nil, nil)
	require.NoError(t, err)
	assert.True(t, resVR.IsVR)
}

func TestEnricher_BangoDmmKeyFallsBackToFilenameBango(t *testing.T) {
	t.Parallel()

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
				assert.Equal(t, "PRED00374", javID, "dmm-derived search key")
				return nil, errors.New("remote timeout")
			},
		},
		{
			name: "empty result",
			mockFunc: func(ctx context.Context, javID string) (map[string]interface{}, error) {
				assert.Equal(t, "PRED00374", javID, "dmm-derived search key")
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			enricher := NewEnricher(&mockMCPClient{searchJapanesePornFunc: tt.mockFunc}, nil, nil)

			res, err := enricher.Enrich(
				context.Background(),
				model.CategoryBangoPorn,
				[]string{"PRED-374.mp4"},
				map[string]interface{}{"dmm_id": "pred00374"},
				nil,
			)
			require.NoError(t, err)
			assert.Equal(t, "PRED-374", res.Bango)
		})
	}
}
