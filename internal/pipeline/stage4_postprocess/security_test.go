package stage4postprocess

import (
	"testing"

	"organizer/internal/model"
)

func mustStrPtr(s string) *string { return &s }

func TestSanitizeRelativeTarget_TraversalBlocked(t *testing.T) {
	malicious := []string{
		"../../etc/passwd",
		"../secret",
		"jav/../../escape",
		"a/../../b",
		"..",
		"/etc/passwd",
		"/tv_series/x",
		"",
		"   ",
	}

	for _, target := range malicious {
		if _, err := SanitizeRelativeTarget(target); err == nil {
			t.Fatalf("target %q must be rejected", target)
		}
	}
}

func TestSanitizeRelativeTarget_ValidPathsCleaned(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tv_series/Chinese/X (2020)/Season 01/X S01E01.mkv", "tv_series/Chinese/X (2020)/Season 01/X S01E01.mkv"},
		{"a/./b//c", "a/b/c"},
		{"jav/素人/SSIS-698.mp4", "jav/素人/SSIS-698.mp4"},
	}

	for _, tt := range tests {
		got, err := SanitizeRelativeTarget(tt.input)
		if err != nil {
			t.Fatalf("target %q unexpectedly rejected: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("SanitizeRelativeTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizePlan_GarbageFilesForcedSkip(t *testing.T) {
	plan := []model.PlanAction{
		{File: "Movie/cover.nfo", Action: "move", Target: mustStrPtr("movie/X/cover.nfo")},
		{File: "Movie/site.url", Action: "move", Target: mustStrPtr("movie/X/site.url")},
		{File: "Movie/movie.torrent", Action: "move", Target: mustStrPtr("movie/X/movie.torrent")},
		{File: "Movie/movie.mkv", Action: "move", Target: mustStrPtr("movie/X/movie.mkv")},
	}

	sanitized := SanitizePlan(plan)
	if len(sanitized) != len(plan) {
		t.Fatalf("sanitize must not drop actions, got %d", len(sanitized))
	}

	for _, file := range []string{"Movie/cover.nfo", "Movie/site.url", "Movie/movie.torrent"} {
		action := findAction(t, sanitized, file)
		if action.Action != "skip" || action.Target != nil {
			t.Fatalf("garbage file %s must be skipped with null target, got %+v", file, action)
		}
	}

	keep := findAction(t, sanitized, "Movie/movie.mkv")
	if keep.Action != "move" || keep.Target == nil || *keep.Target != "movie/X/movie.mkv" {
		t.Fatalf("legitimate file must be kept: %+v", keep)
	}
}

func TestSanitizePlan_MoveWithoutTargetOrEscapingTargetSkipped(t *testing.T) {
	plan := []model.PlanAction{
		{File: "a.mkv", Action: "move"},
		{File: "b.mkv", Action: "move", Target: mustStrPtr("")},
		{File: "c.mkv", Action: "move", Target: mustStrPtr("../../etc/passwd")},
		{File: "d.mkv", Action: "skip"},
		{File: "e.mkv", Action: "move", Target: mustStrPtr("a/./b/../c.mkv")},
	}

	sanitized := SanitizePlan(plan)

	for _, file := range []string{"a.mkv", "b.mkv", "c.mkv"} {
		action := findAction(t, sanitized, file)
		if action.Action != "skip" || action.Target != nil {
			t.Fatalf("file %s must end up skipped with null target, got %+v", file, action)
		}
	}

	skip := findAction(t, sanitized, "d.mkv")
	if skip.Action != "skip" || skip.Target != nil {
		t.Fatalf("skip actions must pass through unchanged: %+v", skip)
	}

	cleaned := findAction(t, sanitized, "e.mkv")
	if cleaned.Action != "move" || cleaned.Target == nil || *cleaned.Target != "a/c.mkv" {
		t.Fatalf("valid target must be cleaned: %+v", cleaned)
	}
}

func findAction(t *testing.T, actions []model.PlanAction, file string) model.PlanAction {
	t.Helper()
	for _, a := range actions {
		if a.File == file {
			return a
		}
	}
	t.Fatalf("action for file %q not found", file)
	return model.PlanAction{}
}
