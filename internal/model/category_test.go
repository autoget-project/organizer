package model

import (
	"encoding/json"
	"testing"
)

func TestISO639ToLanguage(t *testing.T) {
	tests := []struct {
		code     string
		expected Language
	}{
		{"zh", LanguageChinese},
		{"cn", LanguageChinese},
		{"chi", LanguageChinese},
		{"zho", LanguageChinese},
		{"mandarin", LanguageChinese},
		{"cantonese", LanguageChinese},
		{"chinese", LanguageChinese},
		{"en", LanguageEnglish},
		{"eng", LanguageEnglish},
		{"english", LanguageEnglish},
		{"ja", LanguageJapanese},
		{"jp", LanguageJapanese},
		{"jpn", LanguageJapanese},
		{"japanese", LanguageJapanese},
		{"ko", LanguageKorean},
		{"kor", LanguageKorean},
		{"korean", LanguageKorean},
		{"fr", LanguageOthers},
		{"de", LanguageOthers},
		{"unknown", LanguageOthers},
		{"", LanguageOthers},
	}

	for _, tt := range tests {
		got := ISO639ToLanguage(tt.code)
		if got != tt.expected {
			t.Errorf("ISO639ToLanguage(%q) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestCategoryEnums(t *testing.T) {
	if len(AllCategories) != 10 {
		t.Errorf("expected 10 categories, got %d", len(AllCategories))
	}
	if len(SimpleMoveCategories) != 5 {
		t.Errorf("expected 5 simple move categories, got %d", len(SimpleMoveCategories))
	}

	data, err := json.Marshal(CategoryMovie)
	if err != nil {
		t.Fatalf("failed to marshal category: %v", err)
	}
	if string(data) != `"movie"` {
		t.Errorf("unexpected json: %s", string(data))
	}

	var cat Category
	if err := json.Unmarshal([]byte(`"tv_series"`), &cat); err != nil {
		t.Fatalf("failed to unmarshal category: %v", err)
	}
	if cat != CategoryTVSeries {
		t.Errorf("expected tv_series, got %v", cat)
	}
}
