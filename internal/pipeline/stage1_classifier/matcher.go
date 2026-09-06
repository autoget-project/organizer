package stage1classifier

import (
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/autoget-project/organizer/internal/model"
)

// Supported extensions for rule matching.
var (
	bookExtensions = map[string]struct{}{
		".pdf":  {},
		".epub": {},
		".mobi": {},
		".azw3": {},
		".txt":  {},
	}
	audioExtensions = map[string]struct{}{
		".mp3":  {},
		".flac": {},
		".wav":  {},
		".ape":  {},
		".ogg":  {},
		".m4a":  {},
	}
	videoExtensions = map[string]struct{}{
		".mp4": {},
		".mkv": {},
		".avi": {},
		".mov": {},
		".wmv": {},
		".ts":  {},
	}
)

var (
	standardBangoRegex = regexp.MustCompile(`^[A-Za-z]{3,5}-\d{3,4}$`)
	fc2BangoRegex      = regexp.MustCompile(`^(?i:FC2|FC2-PPV)-\d+$`)
)

// MatchByRules performs Stage 1 rule-based classification according to M8 ordering.
// Returns (result, matched) where matched is true if high-confidence rule was hit.
func MatchByRules(files []string, metadata map[string]interface{}) (model.ClassifierResult, bool) {
	// 1. organizer_category fault-tolerant parsing (M8a)
	if metadata != nil {
		if rawVal, ok := metadata["organizer_category"]; ok && rawVal != nil {
			if cat, ok := parseOrganizerCategory(rawVal); ok {
				return model.ClassifierResult{
					Category: cat,
					NeedLLM:  false,
				}, true
			}
			log.Printf("organizer_category provided but no valid Category found in: %v", rawVal)
		}
	}

	// 2. Clear external metadata identifier: dmm_id -> bango_porn
	if metadata != nil {
		if dmmID, ok := metadata["dmm_id"]; ok && dmmID != nil {
			dmmStr := strings.TrimSpace(toString(dmmID))
			if dmmStr != "" {
				entities := map[string]interface{}{
					"dmm_id": dmmStr,
				}
				return model.ClassifierResult{
					Category: model.CategoryBangoPorn,
					NeedLLM:  false,
					Entities: entities,
				}, true
			}
		}
	}

	if len(files) == 0 {
		return model.ClassifierResult{}, false
	}

	// 3. Pure eBook extensions -> book
	if allMatchExtensions(files, bookExtensions) {
		return model.ClassifierResult{
			Category: model.CategoryBook,
			NeedLLM:  false,
		}, true
	}

	// 4. Pure audio disambiguation degradation (M8b Plan A)
	// When all files are audio, do NOT blindly return music! Return unmatched to let LLM disambiguate.
	if allMatchExtensions(files, audioExtensions) {
		return model.ClassifierResult{}, false
	}

	// 5. Strong feature naming:
	// Single main video file and filename precisely matches anchored standard bango regex.
	mainVideos := filterMainVideos(files)
	if len(mainVideos) == 1 {
		base := filepath.Base(mainVideos[0])
		ext := filepath.Ext(base)
		stem := strings.TrimSuffix(base, ext)

		if standardBangoRegex.MatchString(stem) || fc2BangoRegex.MatchString(stem) {
			entities := map[string]interface{}{
				"bango": strings.ToUpper(stem),
			}
			return model.ClassifierResult{
				Category: model.CategoryBangoPorn,
				NeedLLM:  false,
				Entities: entities,
			}, true
		}
	}

	// 6. No rule hit -> fallback to LLM
	return model.ClassifierResult{}, false
}

func parseOrganizerCategory(raw any) (model.Category, bool) {
	switch v := raw.(type) {
	case model.Category:
		if isValidCategory(v) {
			return v, true
		}
	case string:
		cat := model.Category(strings.TrimSpace(v))
		if isValidCategory(cat) {
			return cat, true
		}
	case []model.Category:
		for _, item := range v {
			if isValidCategory(item) {
				return item, true
			}
		}
	case []string:
		for _, item := range v {
			cat := model.Category(strings.TrimSpace(item))
			if isValidCategory(cat) {
				return cat, true
			}
		}
	case []interface{}:
		for _, item := range v {
			if strVal, ok := item.(string); ok {
				cat := model.Category(strings.TrimSpace(strVal))
				if isValidCategory(cat) {
					return cat, true
				}
			} else if catVal, ok := item.(model.Category); ok {
				if isValidCategory(catVal) {
					return catVal, true
				}
			}
		}
	}
	return model.CategoryUnknown, false
}

func isValidCategory(c model.Category) bool {
	for _, recognized := range model.AllCategories {
		if recognized != model.CategoryUnknown && recognized == c {
			return true
		}
	}
	return false
}

func allMatchExtensions(files []string, extMap map[string]struct{}) bool {
	if len(files) == 0 {
		return false
	}
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if _, ok := extMap[ext]; !ok {
			return false
		}
	}
	return true
}

func filterMainVideos(files []string) []string {
	var videos []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if _, ok := videoExtensions[ext]; ok {
			videos = append(videos, f)
		}
	}
	return videos
}

func toString(val any) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}
