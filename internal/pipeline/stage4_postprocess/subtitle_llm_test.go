package stage4postprocess

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

func TestReadSubtitlePreview_First30LinesAndPathJoin_L8(t *testing.T) {
	downloadDir := t.TempDir()

	// L8: the preview must be read from DOWNLOAD_COMPLETED_DIR/{dir}/{file}.
	content := make([]string, 0, 40)
	for i := 1; i <= 40; i++ {
		content = append(content, "line-"+strings.Repeat("x", i%3)+"-"+string(rune('A'+i%26)))
	}
	full := filepath.Join(downloadDir, "movie1", "sub.srt")
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(strings.Join(content, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	preview := ReadSubtitlePreview(downloadDir, "movie1", "sub.srt")
	lines := strings.Split(preview, "\n")
	if len(lines) != SubtitlePreviewLines {
		t.Fatalf("expected %d lines, got %d", SubtitlePreviewLines, len(lines))
	}
	if strings.Contains(preview, content[39]) {
		t.Fatalf("preview must not contain lines beyond the first 30")
	}

	// Missing files degrade to an error marker instead of crashing.
	if preview := ReadSubtitlePreview(downloadDir, "movie1", "missing.srt"); !strings.Contains(preview, "Error reading file") {
		t.Fatalf("expected error marker for missing file, got %q", preview)
	}
}

func TestSubtitleTargetPath_JapaneseStrictlyJpn(t *testing.T) {
	videoTarget := "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.mkv"

	tests := []struct {
		name string
		lang model.Language
		want string
	}{
		{
			name: "japanese uses jpn",
			lang: model.LanguageJapanese,
			want: "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.日本語.jpn.srt",
		},
		{
			name: "chinese uses chi",
			lang: model.LanguageChinese,
			want: "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.简体中文.chi.srt",
		},
		{
			name: "english uses eng",
			lang: model.LanguageEnglish,
			want: "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.English.eng.srt",
		},
		{
			name: "unknown keeps plain base name",
			lang: model.LanguageOthers,
			want: "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.srt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubtitleTargetPath(videoTarget, tt.lang, ".srt")
			if got != tt.want {
				t.Fatalf("SubtitleTargetPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubtitlePlanner_PairingWithJpnNaming(t *testing.T) {
	downloadDir := t.TempDir()
	subDir := filepath.Join(downloadDir, "anime1")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	jpnSub := "1\n00:00:01,000 --> 00:00:02,000\nありがとう\n"
	if err := os.WriteFile(filepath.Join(subDir, "ep1.jp.srt"), []byte(jpnSub), 0644); err != nil {
		t.Fatal(err)
	}

	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "subtitle files to match their corresponding video",
		Response: `{"plan":[
			{"file":"ep1.jp.srt","action":"move","matched_video":"ep1.mkv","language":"Japanese"}
		]}`,
	})

	planner := NewSubtitlePlanner(prov, downloadDir)
	videoPlan := []model.PlanAction{
		{
			File:   "ep1.mkv",
			Action: "move",
			Target: strPtr("anim_tv_series/Japanese/某动画 (2023)/Season 01/某动画 (2023) S01E01.mkv"),
		},
	}

	actions, err := planner.PairSubtitles(context.Background(), "anime1", []string{"ep1.jp.srt"}, videoPlan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	action := findAction(t, actions, "ep1.jp.srt")
	if action.Action != "move" || action.Target == nil {
		t.Fatalf("expected subtitle move, got %+v", action)
	}
	want := "anim_tv_series/Japanese/某动画 (2023)/Season 01/某动画 (2023) S01E01.日本語.jpn.srt"
	if *action.Target != want {
		t.Fatalf("subtitle target = %q, want %q", *action.Target, want)
	}

	// The prompt must embed the first-30-lines preview read via the dir join.
	calls := prov.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 provider call, got %d", len(calls))
	}
	if !strings.Contains(calls[0].Prompt, "ありがとう") {
		t.Fatalf("prompt must embed the subtitle content preview")
	}
}

func TestSubtitlePlanner_UnmatchedAndUncoveredSkipped(t *testing.T) {
	downloadDir := t.TempDir()
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "subtitle files to match their corresponding video",
		Response: `{"plan":[
			{"file":"d1/orphan.srt","action":"skip"},
			{"file":"d1/ghost.srt","action":"move","matched_video":"d1/not-in-plan.mkv","language":"Chinese"}
		]}`,
	})

	planner := NewSubtitlePlanner(prov, downloadDir)
	videoPlan := []model.PlanAction{
		{File: "d1/movie.mkv", Action: "move", Target: strPtr("movie/X (2000)/X (2000).mkv")},
	}

	actions, err := planner.PairSubtitles(context.Background(), "d1", []string{"d1/orphan.srt", "d1/ghost.srt", "d1/uncovered.srt"}, videoPlan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	orphan := findAction(t, actions, "d1/orphan.srt")
	if orphan.Action != "skip" {
		t.Fatalf("expected orphan skip, got %+v", orphan)
	}
	ghost := findAction(t, actions, "d1/ghost.srt")
	if ghost.Action != "skip" {
		t.Fatalf("matched video outside the plan must be skipped, got %+v", ghost)
	}
	uncovered := findAction(t, actions, "d1/uncovered.srt")
	if uncovered.Action != "skip" {
		t.Fatalf("uncovered subtitle must be skipped, got %+v", uncovered)
	}
}
