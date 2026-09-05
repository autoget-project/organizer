package stage2enricher

import (
	"context"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"organizer/internal/ai"
	"organizer/internal/mcp"
	"organizer/internal/model"
)

var (
	yearRegex  = regexp.MustCompile(`\b(19\d\d|20\d\d)\b`)
	bangoRegex = regexp.MustCompile(`(?i)([A-Z]{2,5}-\d{2,7}|FC2(?:-PPV)?-\d{4,8}|(?:MD|MDCM|MDHG|MDHT|MDL|MDSR|MSD)-\d{2,7})`)
)

// madouLabels mirrors the Python prompt's exact prefix set
// (archived/app/agents/categorizer/is_bango_porn.py).
var madouLabels = map[string]struct{}{
	"MD": {}, "MDCM": {}, "MDHG": {}, "MDHT": {}, "MDL": {}, "MDSR": {}, "MSD": {},
}

// isMadouBango reports whether the bango's hyphen-prefixed label belongs to the
// exact Madou label set. A bare HasPrefix("MD") would misclassify mainstream
// JAV labels such as MIDE/MIDD/MDBK/MDYD.
func isMadouBango(bango string) bool {
	prefix, _, _ := strings.Cut(strings.ToUpper(bango), "-")
	_, ok := madouLabels[prefix]
	return ok
}

// Enricher orchestrates Stage 2 metadata retrieval and degradation protection (M6).
type Enricher struct {
	mcpClient  mcp.Client
	actorStore *ActorStore
	aiProvider ai.Provider
}

// NewEnricher creates a new Enricher instance.
func NewEnricher(mcpClient mcp.Client, actorStore *ActorStore, aiProvider ai.Provider) *Enricher {
	return &Enricher{
		mcpClient:  mcpClient,
		actorStore: actorStore,
		aiProvider: aiProvider,
	}
}

// Enrich enriches metadata according to media Category, applying graceful degradation on failures (M6).
func (e *Enricher) Enrich(ctx context.Context, cat model.Category, files []string, metadata map[string]interface{}, entities map[string]interface{}) (model.EnrichedMetadata, error) {
	switch cat {
	case model.CategoryMovie:
		return e.enrichMovie(ctx, files, metadata, entities)
	case model.CategoryTVSeries:
		return e.enrichTVSeries(ctx, files, metadata, entities)
	case model.CategoryBangoPorn:
		return e.enrichBangoPorn(ctx, files, metadata, entities)
	case model.CategoryPorn:
		return e.enrichPorn(ctx, files, metadata, entities)
	default:
		// simple categories (book, music, photobook, audio_book, music_video) or unknown: skip Stage 2
		return model.EnrichedMetadata{
			Language: model.LanguageOthers,
		}, nil
	}
}

func (e *Enricher) enrichMovie(ctx context.Context, files []string, metadata map[string]interface{}, entities map[string]interface{}) (model.EnrichedMetadata, error) {
	var enriched model.EnrichedMetadata
	enriched.Language = model.LanguageOthers

	imdbID := getIMDbID(metadata, entities)
	titleCandidate := getTitleCandidate(files, metadata, entities)

	// Step 1: Try find_by_imdb_id if imdbID is present
	var movieFound bool
	if imdbID != "" && e.mcpClient != nil {
		res, err := e.mcpClient.FindByIMDbID(ctx, imdbID)
		if err == nil && res != nil {
			if movieResults, ok := res["movie_results"].([]interface{}); ok && len(movieResults) > 0 {
				if m, ok := movieResults[0].(map[string]interface{}); ok {
					movieFound = true
					e.populateMovieFromMap(&enriched, m)
				}
			}
		} else {
			log.Printf("[M6 degrade] find_by_imdb_id failed for movie (%s): %v, falling back to title search", imdbID, err)
		}
	}

	// Step 2: Fallback to search_movies by title
	if !movieFound && titleCandidate != "" && e.mcpClient != nil {
		res, err := e.mcpClient.SearchMovies(ctx, titleCandidate)
		if err == nil && res != nil {
			if results, ok := res["results"].([]interface{}); ok && len(results) > 0 {
				if m, ok := results[0].(map[string]interface{}); ok {
					movieFound = true
					e.populateMovieFromMap(&enriched, m)
				}
			}
		} else {
			log.Printf("[M6 degrade] search_movies failed for (%s): %v", titleCandidate, err)
		}
	}

	// Step 3: Final fallback using cleaned name from stage 1 or filenames
	if !movieFound {
		if enriched.Title == "" {
			enriched.Title = titleCandidate
		}
		if enriched.Year == 0 {
			enriched.Year = extractYear(files, metadata)
		}
	}

	// Double-check is_anim indicators
	if !enriched.IsAnim {
		enriched.IsAnim = detectIsAnim(files, metadata, enriched.Title)
	}

	return enriched, nil
}

func (e *Enricher) enrichTVSeries(ctx context.Context, files []string, metadata map[string]interface{}, entities map[string]interface{}) (model.EnrichedMetadata, error) {
	var enriched model.EnrichedMetadata
	enriched.Language = model.LanguageOthers

	imdbID := getIMDbID(metadata, entities)
	titleCandidate := getTitleCandidate(files, metadata, entities)

	var tvFound bool
	if imdbID != "" && e.mcpClient != nil {
		res, err := e.mcpClient.FindByIMDbID(ctx, imdbID)
		if err == nil && res != nil {
			if tvResults, ok := res["tv_results"].([]interface{}); ok && len(tvResults) > 0 {
				if t, ok := tvResults[0].(map[string]interface{}); ok {
					tvFound = true
					e.populateTVFromMap(&enriched, t)
				}
			}
		} else {
			log.Printf("[M6 degrade] find_by_imdb_id failed for tv_series (%s): %v, falling back to title search", imdbID, err)
		}
	}

	// Fallback to search_tv_shows
	if !tvFound && titleCandidate != "" && e.mcpClient != nil {
		res, err := e.mcpClient.SearchTVShows(ctx, titleCandidate)
		if err == nil && res != nil {
			if results, ok := res["results"].([]interface{}); ok && len(results) > 0 {
				if t, ok := results[0].(map[string]interface{}); ok {
					tvFound = true
					e.populateTVFromMap(&enriched, t)
				}
			}
		} else {
			log.Printf("[M6 degrade] search_tv_shows failed for (%s): %v", titleCandidate, err)
		}
	}

	if !tvFound {
		if enriched.Title == "" {
			enriched.Title = titleCandidate
		}
		if enriched.Year == 0 {
			enriched.Year = extractYear(files, metadata)
		}
	}

	if !enriched.IsAnim {
		enriched.IsAnim = detectIsAnim(files, metadata, enriched.Title)
	}

	return enriched, nil
}

func (e *Enricher) enrichBangoPorn(ctx context.Context, files []string, metadata map[string]interface{}, entities map[string]interface{}) (model.EnrichedMetadata, error) {
	var enriched model.EnrichedMetadata
	enriched.Language = model.LanguageJapanese

	// The dmm-derived candidate only serves as the MCP search key; per spec M6
	// the final bango must be the canonical hyphenated form derived from the
	// filename whenever the search fails or returns nothing.
	searchKey := getBangoCandidate(files, metadata, entities)
	bangoCandidate := searchKey

	// Search JAV info via search_japanese_porn
	if searchKey != "" && e.mcpClient != nil {
		res, err := e.mcpClient.SearchJapanesePorn(ctx, searchKey)
		if err == nil && res != nil {
			if actresses, ok := res["actresses"].([]interface{}); ok {
				for _, a := range actresses {
					if actStr, ok := a.(string); ok && strings.TrimSpace(actStr) != "" {
						enriched.Actors = append(enriched.Actors, strings.TrimSpace(actStr))
					}
				}
			}
			if maker, ok := res["maker"].(string); ok {
				enriched.Maker = maker
			}
			if isVR, ok := res["is_vr"].(bool); ok && isVR {
				enriched.IsVR = true
			}
			if title, ok := res["title"].(string); ok {
				enriched.Title = title
			}
		} else {
			log.Printf("[M6 degrade] search_japanese_porn failed for (%s): %v", searchKey, err)
			// Spec M6: fall back to the filename-derived canonical bango.
			if fb := bangoFromFiles(files); fb != "" {
				log.Printf("[M6 degrade] falling back to filename-derived bango (%s)", fb)
				bangoCandidate = fb
			}
		}
	}
	enriched.Bango = bangoCandidate

	// Check Madou prefix against the exact label set (on the final bango).
	if isMadouBango(bangoCandidate) {
		enriched.FromMadou = true
		enriched.Language = model.LanguageChinese
	}

	// If no actresses found, check entities or metadata
	if len(enriched.Actors) == 0 {
		if actorArr, ok := entities["actors"].([]string); ok && len(actorArr) > 0 {
			enriched.Actors = actorArr
		}
	}

	// If VR not flagged, check filename / bango prefix
	if !enriched.IsVR {
		upperBango := strings.ToUpper(bangoCandidate)
		if strings.Contains(upperBango, "VR") {
			enriched.IsVR = true
		}
		for _, f := range files {
			if strings.Contains(strings.ToUpper(f), "VR") {
				enriched.IsVR = true
				break
			}
		}
	}

	// ActorStore directory resolution & maintenance
	if e.actorStore != nil {
		if len(enriched.Actors) > 0 {
			actorDir, err := e.actorStore.SearchAndEnrichActor(ctx, enriched.Actors)
			if err == nil && actorDir != "" {
				// Store the chosen actor dir as the first element or in Actors
				enriched.Actors = append([]string{actorDir}, enriched.Actors...)
			}
		}
	}

	return enriched, nil
}

func (e *Enricher) enrichPorn(ctx context.Context, files []string, metadata map[string]interface{}, entities map[string]interface{}) (model.EnrichedMetadata, error) {
	var enriched model.EnrichedMetadata
	enriched.Language = model.LanguageEnglish

	titleCandidate := getTitleCandidate(files, metadata, entities)
	enriched.Title = titleCandidate

	for _, f := range files {
		if strings.Contains(strings.ToUpper(f), "VR") {
			enriched.IsVR = true
			break
		}
	}
	return enriched, nil
}

func (e *Enricher) populateMovieFromMap(enriched *model.EnrichedMetadata, m map[string]interface{}) {
	if title, ok := m["title"].(string); ok && title != "" {
		enriched.Title = title
	}
	if origTitle, ok := m["original_title"].(string); ok && origTitle != "" {
		enriched.OriginalTitle = origTitle
	}
	if releaseDate, ok := m["release_date"].(string); ok && len(releaseDate) >= 4 {
		if y, err := strconv.Atoi(releaseDate[:4]); err == nil {
			enriched.Year = y
		}
	}
	if lang, ok := m["original_language"].(string); ok && lang != "" {
		enriched.Language = model.ISO639ToLanguage(lang)
	}
	if genres, ok := m["genres"].([]interface{}); ok {
		for _, g := range genres {
			if gMap, ok := g.(map[string]interface{}); ok {
				if gName, ok := gMap["name"].(string); ok {
					if strings.Contains(strings.ToLower(gName), "anim") {
						enriched.IsAnim = true
					}
				}
			} else if gStr, ok := g.(string); ok {
				if strings.Contains(strings.ToLower(gStr), "anim") {
					enriched.IsAnim = true
				}
			}
		}
	}
}

func (e *Enricher) populateTVFromMap(enriched *model.EnrichedMetadata, t map[string]interface{}) {
	if name, ok := t["name"].(string); ok && name != "" {
		enriched.Title = name
	} else if title, ok := t["title"].(string); ok && title != "" {
		enriched.Title = title
	}
	if origName, ok := t["original_name"].(string); ok && origName != "" {
		enriched.OriginalTitle = origName
	}
	if airDate, ok := t["first_air_date"].(string); ok && len(airDate) >= 4 {
		if y, err := strconv.Atoi(airDate[:4]); err == nil {
			enriched.Year = y
		}
	}
	if lang, ok := t["original_language"].(string); ok && lang != "" {
		enriched.Language = model.ISO639ToLanguage(lang)
	}
	if genres, ok := t["genres"].([]interface{}); ok {
		for _, g := range genres {
			if gMap, ok := g.(map[string]interface{}); ok {
				if gName, ok := gMap["name"].(string); ok {
					if strings.Contains(strings.ToLower(gName), "anim") {
						enriched.IsAnim = true
					}
				}
			} else if gStr, ok := g.(string); ok {
				if strings.Contains(strings.ToLower(gStr), "anim") {
					enriched.IsAnim = true
				}
			}
		}
	}
}

func getIMDbID(metadata map[string]interface{}, entities map[string]interface{}) string {
	if entities != nil {
		if id, ok := entities["imdb_id"].(string); ok && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	if metadata != nil {
		if id, ok := metadata["imdb_id"].(string); ok && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func getTitleCandidate(files []string, metadata map[string]interface{}, entities map[string]interface{}) string {
	if entities != nil {
		if t, ok := entities["clean_title"].(string); ok && strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
	}
	if metadata != nil {
		if t, ok := metadata["title"].(string); ok && strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
	}
	if len(files) > 0 {
		base := filepath.Base(files[0])
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return ""
}

func getBangoCandidate(files []string, metadata map[string]interface{}, entities map[string]interface{}) string {
	if entities != nil {
		if b, ok := entities["bango"].(string); ok && strings.TrimSpace(b) != "" {
			return strings.ToUpper(strings.TrimSpace(b))
		}
		if dmm, ok := entities["dmm_id"].(string); ok && strings.TrimSpace(dmm) != "" {
			return strings.ToUpper(strings.TrimSpace(dmm))
		}
	}
	if metadata != nil {
		if dmm, ok := metadata["dmm_id"].(string); ok && strings.TrimSpace(dmm) != "" {
			return strings.ToUpper(strings.TrimSpace(dmm))
		}
	}
	return bangoFromFiles(files)
}

// bangoFromFiles extracts the canonical hyphenated bango from the filenames.
func bangoFromFiles(files []string) string {
	for _, f := range files {
		match := bangoRegex.FindString(filepath.Base(f))
		if match != "" {
			return strings.ToUpper(match)
		}
	}
	return ""
}

func extractYear(files []string, metadata map[string]interface{}) int {
	if metadata != nil {
		if y, ok := metadata["year"].(int); ok && y > 0 {
			return y
		}
		if y, ok := metadata["year"].(float64); ok && y > 0 {
			return int(y)
		}
	}
	for _, f := range files {
		matches := yearRegex.FindAllString(f, -1)
		for _, m := range matches {
			if y, err := strconv.Atoi(m); err == nil && y >= 1950 && y <= 2035 {
				return y
			}
		}
	}
	return 0
}

func detectIsAnim(files []string, metadata map[string]interface{}, title string) bool {
	checkList := []string{title}
	if metadata != nil {
		if desc, ok := metadata["description"].(string); ok {
			checkList = append(checkList, desc)
		}
		if genre, ok := metadata["genre"].(string); ok {
			checkList = append(checkList, genre)
		}
	}
	checkList = append(checkList, files...)

	for _, text := range checkList {
		lower := strings.ToLower(text)
		if strings.Contains(lower, "anime") || strings.Contains(lower, "animation") ||
			strings.Contains(lower, "动画") || strings.Contains(lower, "動漫") ||
			strings.Contains(lower, "新番") {
			return true
		}
	}
	return false
}
