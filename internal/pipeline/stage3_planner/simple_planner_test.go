package stage3planner

import (
	"strings"
	"testing"

	"organizer/internal/model"
)

// planActionsEqual compares two plans ignoring order.
func planActionsEqual(a, b []model.PlanAction) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, action := range a {
		found := false
		for i, other := range b {
			if used[i] {
				continue
			}
			if action.File == other.File && action.Action == other.Action {
				if (action.Target == nil) != (other.Target == nil) {
					continue
				}
				if action.Target != nil && *action.Target != *other.Target {
					continue
				}
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func mustTarget(s string) *string { return &s }

// Ported from archived/app/agents/mover/simple_mover_test.py.
func TestSimplePlanSingleFile(t *testing.T) {
	got := SimplePlan(model.CategoryMovie, []string{"torrent_hash/movie.mp4"})
	want := []model.PlanAction{
		{File: "torrent_hash/movie.mp4", Action: "move", Target: mustTarget("movie/movie.mp4")},
	}
	if !planActionsEqual(got, want) {
		t.Fatalf("unexpected plan: %+v", got)
	}
}

// Ported: branch 2 (files directly under the hash dir -> whole dir move).
func TestSimplePlanMultipleFilesUnderHashDir(t *testing.T) {
	got := SimplePlan(model.CategoryTVSeries, []string{"torrent_hash/episode1.mp4", "torrent_hash/episode2.mp4"})
	want := []model.PlanAction{
		{File: "torrent_hash", Action: "move", Target: mustTarget("tv_series/torrent_hash")},
	}
	if !planActionsEqual(got, want) {
		t.Fatalf("unexpected plan: %+v", got)
	}
}

// Ported: branch 3 (M7) - sub dirs archived with the hash layer stripped.
func TestSimplePlanFilesInSubDirs_Branch3_M7(t *testing.T) {
	got := SimplePlan(model.CategoryBook, []string{
		"torrent_hash/chapter1/page1.pdf",
		"torrent_hash/chapter1/page2.pdf",
		"torrent_hash/chapter2/page1.pdf",
	})
	want := []model.PlanAction{
		{File: "torrent_hash/chapter1", Action: "move", Target: mustTarget("book/chapter1")},
		{File: "torrent_hash/chapter2", Action: "move", Target: mustTarget("book/chapter2")},
	}
	if !planActionsEqual(got, want) {
		t.Fatalf("unexpected plan: %+v", got)
	}

	// Explicit M7 assertion: file = {hash_dir}/{d}, target = {category}/{d}.
	for _, action := range got {
		if action.Target == nil || strings.Contains(*action.Target, "torrent_hash") {
			t.Fatalf("branch 3 target must not keep the hash dir: %v", action.Target)
		}
	}
}

// Ported: mixed files and sub dirs under the hash dir -> whole dir move.
func TestSimplePlanMixedFilesAndDirsUnderHashDir(t *testing.T) {
	got := SimplePlan(model.CategoryMusic, []string{
		"torrent_hash/song.mp3",
		"torrent_hash/album_art/cover.jpg",
	})
	want := []model.PlanAction{
		{File: "torrent_hash", Action: "move", Target: mustTarget("music/torrent_hash")},
	}
	if !planActionsEqual(got, want) {
		t.Fatalf("unexpected plan: %+v", got)
	}
}

func TestSimplePlan_AllFiveCategories(t *testing.T) {
	for _, cat := range model.SimpleMoveCategories {
		got := SimplePlan(cat, []string{"hash/item.dat"})
		if len(got) != 1 {
			t.Fatalf("category %s: expected 1 action, got %d", cat, len(got))
		}
		wantPrefix := string(cat) + "/"
		if got[0].Target == nil || !strings.Contains(*got[0].Target, wantPrefix) {
			t.Fatalf("category %s: expected target prefix %s, got %v", cat, wantPrefix, got[0].Target)
		}
	}
}
