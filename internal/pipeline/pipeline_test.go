package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
	"organizer/internal/pipeline/stage2_enricher"
)

func TestCreatePlan_TVSeriesFullFlow(t *testing.T) {
	downloadDir := t.TempDir()

	// Sandbox files: dirty video name, companion subtitle and garbage.
	subDir := filepath.Join(downloadDir, "show1")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	subtitleContent := "1\n00:00:01,000 --> 00:00:02,000\n你好世界\n"
	if err := os.WriteFile(filepath.Join(subDir, "show.chs.srt"), []byte(subtitleContent), 0644); err != nil {
		t.Fatal(err)
	}

	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "media categorization assistant",
		Response:      `{"category":"tv_series","reason":"episodic naming","entities":{}}`,
	})
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes TV series downloads",
		Response: `{"plan":[{"file":"[HDSky] 我的剧 S01E01.mkv","action":"move",
			"target":"tv_series/Others/我的剧 (2020)/Season 01/我的剧 (2020) S01E01.mkv"}]}`,
	})
	prov.AddRule(mock.Rule{
		PromptPattern: "subtitle files to match their corresponding video",
		Response: `{"plan":[{"file":"show.chs.srt","action":"move",
			"matched_video":"[HDSky] 我的剧 S01E01.mkv","language":"Chinese"}]}`,
	})

	enricher := stage2enricher.NewEnricher(nil, nil, nil)
	p := NewPipeline(prov, enricher, downloadDir)

	files := []string{
		"[HDSky] 我的剧 S01E01.mkv",
		"show.chs.srt",
		"cover.nfo",
	}
	metadata := map[string]interface{}{
		"title": "我的剧",
		"year":  2020.0,
	}

	resp, err := p.CreatePlan(context.Background(), "show1", files, metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("normal planning must keep error null, got %s", *resp.Error)
	}

	actionsByFile := map[string]model.PlanAction{}
	for _, a := range resp.Plan {
		actionsByFile[a.File] = a
	}

	video := actionsByFile["[HDSky] 我的剧 S01E01.mkv"]
	if video.Action != "move" || video.Target == nil ||
		*video.Target != "tv_series/Others/我的剧 (2020)/Season 01/我的剧 (2020) S01E01.mkv" {
		t.Fatalf("unexpected video action: %+v", video)
	}

	sub := actionsByFile["show.chs.srt"]
	if sub.Action != "move" || sub.Target == nil ||
		*sub.Target != "tv_series/Others/我的剧 (2020)/Season 01/我的剧 (2020) S01E01.简体中文.chi.srt" {
		t.Fatalf("unexpected subtitle action: %+v", sub)
	}

	garbage := actionsByFile["cover.nfo"]
	if garbage.Action != "skip" || garbage.Target != nil {
		t.Fatalf("garbage must be skipped with null target, got %+v", garbage)
	}
}

func TestCreatePlan_SubtitlePairingFailureDegradesGracefully(t *testing.T) {
	downloadDir := t.TempDir()
	subDir := filepath.Join(downloadDir, "movie1")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.srt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "media categorization assistant",
		Response:      `{"category":"movie","reason":"single feature","entities":{}}`,
	})
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes movie downloads",
		Response: `{"plan":[{"file":"movie.mkv","action":"move",
			"target":"movie/Others/电影 (2000)/电影 (2000).mkv"}]}`,
	})
	prov.AddRule(mock.Rule{
		PromptPattern: "subtitle files to match their corresponding video",
		Error:         errors.New("subtitle pairing offline"),
	})

	enricher := stage2enricher.NewEnricher(nil, nil, nil)
	p := NewPipeline(prov, enricher, downloadDir)

	resp, err := p.CreatePlan(context.Background(), "movie1",
		[]string{"movie.mkv", "sub.srt"},
		map[string]interface{}{"title": "电影", "year": 2000.0})
	if err != nil {
		t.Fatalf("subtitle pairing failure must degrade, not fail: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("error must stay null on degradation, got %s", *resp.Error)
	}

	videoFound, subFound := false, false
	for _, a := range resp.Plan {
		if a.File == "movie.mkv" && a.Action == "move" {
			videoFound = true
		}
		if a.File == "sub.srt" {
			subFound = true
		}
	}
	if !videoFound {
		t.Fatalf("video plan must survive subtitle failure")
	}
	if subFound {
		t.Fatalf("subtitle must be left unplanned after pairing failure")
	}
}

func TestCreatePlan_UnknownCategoryEmptyPlan(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "media categorization assistant",
		Response:      `{"category":"unknown","reason":"random junk","entities":{}}`,
	})

	p := NewPipeline(prov, stage2enricher.NewEnricher(nil, nil, nil), t.TempDir())

	resp, err := p.CreatePlan(context.Background(), "junk", []string{"junk.bin"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("error must stay null, got %s", *resp.Error)
	}
	if len(resp.Plan) != 0 {
		t.Fatalf("unknown category must return an empty plan, got %+v", resp.Plan)
	}
}

func TestCreatePlan_Stage3LLMFailureIsFatal(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "media categorization assistant",
		Response:      `{"category":"tv_series","reason":"episodic naming","entities":{}}`,
	})
	// No tv planner rule -> mock provider returns its default error.

	p := NewPipeline(prov, stage2enricher.NewEnricher(nil, nil, nil), t.TempDir())

	_, err := p.CreatePlan(context.Background(), "show", []string{"show/ep01.mkv"}, nil)
	if err == nil {
		t.Fatalf("stage 3 LLM failure must surface as a fatal error")
	}
}
