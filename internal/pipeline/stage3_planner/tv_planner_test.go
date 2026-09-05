package stage3planner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

// findAction returns the action for the given file.
func findAction(t *testing.T, actions []model.PlanAction, file string) model.PlanAction {
	t.Helper()
	for _, a := range actions {
		if a.File == file {
			return a
		}
	}
	t.Fatalf("action for file %q not found in plan", file)
	return model.PlanAction{}
}

func TestTVPlanner_AnimRouting_H4(t *testing.T) {
	tests := []struct {
		name          string
		isAnim        bool
		wantPromptTip string
		wantTargetTip string
	}{
		{
			name:          "anim goes to anim_tv_series",
			isAnim:        true,
			wantPromptTip: "anim_tv_series/Chinese",
			wantTargetTip: "anim_tv_series/",
		},
		{
			name:          "non-anim goes to tv_series",
			isAnim:        false,
			wantPromptTip: "tv_series/Chinese",
			wantTargetTip: "tv_series/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov := mock.NewProvider()
			prov.AddRule(mock.Rule{
				PromptPattern: "organizes TV series downloads",
				Response: `{"plan":[{"file":"Show/Show S01E01.mkv","action":"move",
					"target":"` + tt.wantTargetTip + `Chinese/我和僵尸有个约会 (1998)/Season 02/我和僵尸有个约会 (1998) S02E01.mkv"}]}`,
			})

			planner := NewTVPlanner(prov)
			pc := &PlannerContext{
				Files: []string{"Show/Show S01E01.mkv"},
				Metadata: model.EnrichedMetadata{
					Title:    "我和僵尸有个约会",
					Year:     1998,
					IsAnim:   tt.isAnim,
					Language: model.LanguageChinese,
				},
			}

			actions, err := planner.Plan(context.Background(), pc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The planner must inject the H4 root into the LLM prompt.
			calls := prov.Calls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 provider call, got %d", len(calls))
			}
			if !strings.Contains(calls[0].Prompt, tt.wantPromptTip) {
				t.Fatalf("prompt root_path mismatch: want tip %q in prompt", tt.wantPromptTip)
			}

			action := findAction(t, actions, "Show/Show S01E01.mkv")
			if action.Action != "move" {
				t.Fatalf("expected move action, got %s", action.Action)
			}
			if action.Target == nil || !strings.HasPrefix(*action.Target, tt.wantTargetTip) {
				t.Fatalf("expected target starting with %s, got %v", tt.wantTargetTip, action.Target)
			}
		})
	}
}

func TestTVPlanner_MessyEpisodeMapping(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes TV series downloads",
		Response: `{"plan":[
			{"file":"Show/第03话.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 01/X (2020) S01E03.mkv"},
			{"file":"Show/1x05.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 01/X (2020) S01E05.mkv"},
			{"file":"Show/[01-02合集] E01.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.mkv"},
			{"file":"Show/OVA1.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 00/X (2020) S00E01.mkv"},
			{"file":"Show/SP02.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 00/X (2020) S00E02.mkv"},
			{"file":"Show/E01v2.mkv","action":"move","target":"tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.mkv"},
			{"file":"Show/sample.mkv","action":"skip"}
		]}`,
	})

	planner := NewTVPlanner(prov)
	pc := &PlannerContext{
		Files: []string{
			"Show/第03话.mkv",
			"Show/1x05.mkv",
			"Show/[01-02合集] E01.mkv",
			"Show/OVA1.mkv",
			"Show/SP02.mkv",
			"Show/E01v2.mkv",
			"Show/sample.mkv",
			"Show/cover.jpg",
		},
		Metadata: model.EnrichedMetadata{Title: "X", Year: 2020, Language: model.LanguageChinese},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every input file must be accounted for exactly once.
	if len(actions) != len(pc.Files) {
		t.Fatalf("expected %d actions, got %d", len(pc.Files), len(actions))
	}

	wantTargets := map[string]string{
		"Show/第03话.mkv":          "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E03.mkv",
		"Show/1x05.mkv":          "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E05.mkv",
		"Show/[01-02合集] E01.mkv": "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.mkv",
		"Show/OVA1.mkv":          "tv_series/Chinese/X (2020)/Season 00/X (2020) S00E01.mkv",
		"Show/SP02.mkv":          "tv_series/Chinese/X (2020)/Season 00/X (2020) S00E02.mkv",
		"Show/E01v2.mkv":         "tv_series/Chinese/X (2020)/Season 01/X (2020) S01E01.mkv",
	}
	for file, want := range wantTargets {
		action := findAction(t, actions, file)
		if action.Action != "move" || action.Target == nil || *action.Target != want {
			t.Fatalf("file %s: want move to %s, got %s %v", file, want, action.Action, action.Target)
		}
	}

	// Sample skipped by the LLM, non-media skipped locally.
	for _, file := range []string{"Show/sample.mkv", "Show/cover.jpg"} {
		action := findAction(t, actions, file)
		if action.Action != "skip" || action.Target != nil {
			t.Fatalf("file %s: want skip with null target, got %s %v", file, action.Action, action.Target)
		}
	}
}

func TestTVPlanner_UncoveredVideoMarkedSkip(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes TV series downloads",
		Response:      `{"plan":[{"file":"Show/E01.mkv","action":"move","target":"tv_series/Others/X (2020)/Season 01/X (2020) S01E01.mkv"}]}`,
	})

	planner := NewTVPlanner(prov)
	pc := &PlannerContext{
		Files:    []string{"Show/E01.mkv", "Show/E02.mkv"},
		Metadata: model.EnrichedMetadata{Title: "X", Year: 2020},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	action := findAction(t, actions, "Show/E02.mkv")
	if action.Action != "skip" {
		t.Fatalf("expected uncovered video to be skipped, got %s", action.Action)
	}
}

func TestTVPlanner_LLMErrorPropagates(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes TV series downloads",
		Error:         fmt.Errorf("provider outage"),
	})

	planner := NewTVPlanner(prov)
	_, err := planner.Plan(context.Background(), &PlannerContext{
		Files:    []string{"Show/E01.mkv"},
		Metadata: model.EnrichedMetadata{Title: "X"},
	})
	if err == nil {
		t.Fatalf("expected LLM failure to propagate as fatal error")
	}
}
