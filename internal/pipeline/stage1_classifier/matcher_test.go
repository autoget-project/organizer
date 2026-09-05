package stage1classifier

import (
	"testing"

	"organizer/internal/model"
)

func TestMatchByRules_OrganizerCategory(t *testing.T) {
	// M8a: organizer_category = ["audio_book"] with pure mp3 files -> audio_book (NOT music)
	reqFiles := []string{"chapter01.mp3", "chapter02.mp3"}
	meta := map[string]interface{}{
		"organizer_category": []string{"audio_book"},
	}
	res, matched := MatchByRules(reqFiles, meta)
	if !matched {
		t.Fatalf("expected matched to be true")
	}
	if res.Category != model.CategoryAudioBook {
		t.Fatalf("expected CategoryAudioBook, got %s", res.Category)
	}

	// Single string organizer_category
	metaStr := map[string]interface{}{
		"organizer_category": "book",
	}
	resStr, matchedStr := MatchByRules([]string{"anything.mp4"}, metaStr)
	if !matchedStr || resStr.Category != model.CategoryBook {
		t.Fatalf("expected CategoryBook, got %v (%t)", resStr.Category, matchedStr)
	}

	// Array with invalid item first, then valid bango_porn
	metaInvalidFirst := map[string]interface{}{
		"organizer_category": []string{"invalid_category_xyz", "bango_porn"},
	}
	resInv, matchedInv := MatchByRules([]string{"video.mp4"}, metaInvalidFirst)
	if !matchedInv || resInv.Category != model.CategoryBangoPorn {
		t.Fatalf("expected CategoryBangoPorn, got %v (%t)", resInv.Category, matchedInv)
	}

	// All invalid items: should not panic/crash, continues evaluation
	metaAllInvalid := map[string]interface{}{
		"organizer_category": []string{"invalid_item_1", "invalid_item_2"},
	}
	resAllInv, matchedAllInv := MatchByRules([]string{"ebook.epub"}, metaAllInvalid)
	// Falls through to pure book check -> book
	if !matchedAllInv || resAllInv.Category != model.CategoryBook {
		t.Fatalf("expected CategoryBook after all invalid organizer_category, got %v (%t)", resAllInv.Category, matchedAllInv)
	}
}

func TestMatchByRules_DmmID(t *testing.T) {
	meta := map[string]interface{}{
		"dmm_id": "ssis00123",
	}
	res, matched := MatchByRules([]string{"random_name.mp4"}, meta)
	if !matched {
		t.Fatalf("expected dmm_id to match")
	}
	if res.Category != model.CategoryBangoPorn {
		t.Fatalf("expected CategoryBangoPorn, got %s", res.Category)
	}
	if res.Entities["dmm_id"] != "ssis00123" {
		t.Fatalf("expected dmm_id in entities")
	}
}

func TestMatchByRules_PureBook(t *testing.T) {
	files := []string{"book1.epub", "folder/book2.pdf", "text.txt"}
	res, matched := MatchByRules(files, nil)
	if !matched {
		t.Fatalf("expected matched book")
	}
	if res.Category != model.CategoryBook {
		t.Fatalf("expected CategoryBook, got %s", res.Category)
	}
}

func TestMatchByRules_PureAudioDegradation(t *testing.T) {
	// M8b: Pure audio files without metadata should NOT blindly match music, should return false for LLM disambiguation
	files := []string{"track01.mp3", "track02.flac", "audio.m4a"}
	_, matched := MatchByRules(files, nil)
	if matched {
		t.Fatalf("expected matched to be false for pure audio without hints (must degrade to LLM)")
	}
}

func TestMatchByRules_StandardBangoRegex(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"SSIS-001.mp4", "SSIS-001"},
		{"abp-123.mkv", "ABP-123"},
		{"FC2-PPV-123456.mp4", "FC2-PPV-123456"},
		{"FC2-1234567.mp4", "FC2-1234567"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			res, matched := MatchByRules([]string{tt.filename}, nil)
			if !matched {
				t.Fatalf("expected bango match for %s", tt.filename)
			}
			if res.Category != model.CategoryBangoPorn {
				t.Fatalf("expected CategoryBangoPorn, got %s", res.Category)
			}
			if res.Entities["bango"] != tt.expected {
				t.Fatalf("expected bango %s, got %v", tt.expected, res.Entities["bango"])
			}
		})
	}
}

func TestMatchByRules_ComplexNoiseFallthrough(t *testing.T) {
	// Release group noise or multiple video files without bango pattern should fall through to false
	files := []string{
		"[HDSky] The.Matrix.1999.1080p.BluRay.x264.mkv",
		"sample.mkv",
	}
	_, matched := MatchByRules(files, nil)
	if matched {
		t.Fatalf("expected false for complex noisy movie files")
	}
}
