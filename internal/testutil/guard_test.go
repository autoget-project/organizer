package testutil_test

import (
	"testing"

	"organizer/internal/testutil"
)

func TestSkipIfNoAPIKey_WhenSet(t *testing.T) {
	t.Setenv("XAI_API_KEY", "dummy-key")
	t.Setenv("GEMINI_API_KEY", "dummy-key")

	// Should not skip
	testutil.SkipIfNoAPIKey(t, "grok")
	testutil.SkipIfNoAPIKey(t, "gemini")
	testutil.SkipIfNoAPIKey(t, "all")
}

func TestSkipIfNoAPIKey_WhenMissing(t *testing.T) {
	t.Setenv("XAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	t.Run("grok", func(t *testing.T) {
		testutil.SkipIfNoAPIKey(t, "grok")
		if !t.Skipped() {
			t.Errorf("expected skip for provider grok when XAI_API_KEY unset")
		}
	})

	t.Run("gemini", func(t *testing.T) {
		testutil.SkipIfNoAPIKey(t, "gemini")
		if !t.Skipped() {
			t.Errorf("expected skip for provider gemini when GEMINI_API_KEY unset")
		}
	})

	t.Run("all", func(t *testing.T) {
		testutil.SkipIfNoAPIKey(t, "all")
		if !t.Skipped() {
			t.Errorf("expected skip for provider all when both keys unset")
		}
	})
}
