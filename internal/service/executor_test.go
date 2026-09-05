package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"organizer/internal/model"
)

func strPtr(s string) *string { return &s }

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestExecutePlan_SingleFileMoveAndArchive(t *testing.T) {
	downloadDir := t.TempDir()
	targetDir := t.TempDir()
	mustWriteFile(t, filepath.Join(downloadDir, "d1", "movie.mkv"), "data")

	exec := NewExecutor(downloadDir, targetDir)
	resp, err := exec.ExecutePlan(context.Background(), "d1", []model.PlanAction{
		{File: "movie.mkv", Action: "move", Target: strPtr("movie/Others/Movie (2000)/Movie (2000).mkv")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.FailedMove) != 0 {
		t.Fatalf("expected empty failed_move, got %+v", resp.FailedMove)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "movie", "Others", "Movie (2000)", "Movie (2000).mkv")); err != nil {
		t.Fatalf("file must be moved to the target library: %v", err)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "d1")); !os.IsNotExist(err) {
		t.Fatalf("source dir must be gone after archiving, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "archive", "d1")); err != nil {
		t.Fatalf("source dir must be archived to archive/d1: %v", err)
	}
}

func TestExecutePlan_DirectoryAtomicMove(t *testing.T) {
	downloadDir := t.TempDir()
	targetDir := t.TempDir()
	mustWriteFile(t, filepath.Join(downloadDir, "d2", "season", "ep01.mkv"), "a")
	mustWriteFile(t, filepath.Join(downloadDir, "d2", "season", "ep02.mkv"), "b")

	exec := NewExecutor(downloadDir, targetDir)
	resp, err := exec.ExecutePlan(context.Background(), "d2", []model.PlanAction{
		{File: "season", Action: "move", Target: strPtr("tv_series/Others/Show (2020)/Season 01")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.FailedMove) != 0 {
		t.Fatalf("directory move must succeed atomically, got %+v", resp.FailedMove)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "tv_series", "Others", "Show (2020)", "Season 01", "ep01.mkv")); err != nil {
		t.Fatalf("directory content must be moved intact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "tv_series", "Others", "Show (2020)", "Season 01", "ep02.mkv")); err != nil {
		t.Fatalf("directory content must be moved intact: %v", err)
	}
}

func TestExecutePlan_MixedFailureAggregationKeepsLegalMoves(t *testing.T) {
	downloadDir := t.TempDir()
	targetDir := t.TempDir()
	mustWriteFile(t, filepath.Join(downloadDir, "d3", "good.mkv"), "data")

	exec := NewExecutor(downloadDir, targetDir)
	resp, err := exec.ExecutePlan(context.Background(), "d3", []model.PlanAction{
		{File: "missing.mkv", Action: "move", Target: strPtr("movie/Others/M (2000)/M (2000).mkv")},
		{File: "good.mkv", Action: "move", Target: strPtr("movie/Others/M (2000)/M (2000) part.2.mkv")},
	})
	if err != nil {
		t.Fatalf("aggregated failure must not surface as a fatal error: %v", err)
	}
	if len(resp.FailedMove) != 1 {
		t.Fatalf("expected exactly one failed move, got %+v", resp.FailedMove)
	}
	failed := resp.FailedMove[0]
	if failed.File != "missing.mkv" || failed.Action != "move" || failed.Reason != "file not found" {
		t.Fatalf("unexpected failed entry: %+v", failed)
	}
	if failed.Target == nil || *failed.Target != "movie/Others/M (2000)/M (2000).mkv" {
		t.Fatalf("failed entry must carry the original target, got %+v", failed.Target)
	}

	// L11: the legal move must still have been executed.
	if _, err := os.Stat(filepath.Join(targetDir, "movie", "Others", "M (2000)", "M (2000) part.2.mkv")); err != nil {
		t.Fatalf("legal move must proceed despite the missing file: %v", err)
	}
	// L11: the source directory must NOT be archived on any failure.
	if _, err := os.Stat(filepath.Join(downloadDir, "d3")); err != nil {
		t.Fatalf("source dir must stay in place when failures exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "archive", "d3")); !os.IsNotExist(err) {
		t.Fatalf("source dir must not be archived when failures exist, stat err: %v", err)
	}
}

func TestExecutePlan_SourceDirAlreadyCleanedSkipsArchive(t *testing.T) {
	downloadDir := t.TempDir()
	targetDir := t.TempDir()

	// The source dir "d4" never existed / was cleaned externally: no move
	// actions, so everything "succeeds" and the archive step must be skipped
	// without any error (L15).
	exec := NewExecutor(downloadDir, targetDir)
	resp, err := exec.ExecutePlan(context.Background(), "d4", []model.PlanAction{
		{File: "note.txt", Action: "skip", Target: nil},
	})
	if err != nil {
		t.Fatalf("missing source dir must not error, got: %v", err)
	}
	if len(resp.FailedMove) != 0 {
		t.Fatalf("expected empty failed_move, got %+v", resp.FailedMove)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "archive", "d4")); !os.IsNotExist(err) {
		t.Fatalf("archive must be skipped when the source dir is gone, stat err: %v", err)
	}
}

func TestExecutePlan_SkipActionsAreIgnored(t *testing.T) {
	downloadDir := t.TempDir()
	targetDir := t.TempDir()
	mustWriteFile(t, filepath.Join(downloadDir, "d5", "cover.nfo"), "junk")

	exec := NewExecutor(downloadDir, targetDir)
	resp, err := exec.ExecutePlan(context.Background(), "d5", []model.PlanAction{
		{File: "cover.nfo", Action: "skip", Target: nil},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.FailedMove) != 0 {
		t.Fatalf("skip actions must not fail, got %+v", resp.FailedMove)
	}
	// Full success (only skips) still archives the source dir with the
	// skipped files inside.
	if _, err := os.Stat(filepath.Join(downloadDir, "archive", "d5", "cover.nfo")); err != nil {
		t.Fatalf("source dir must be archived on full success: %v", err)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "d5")); !os.IsNotExist(err) {
		t.Fatalf("source dir must be gone after archiving, stat err: %v", err)
	}
}
