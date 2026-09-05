package stage3planner

import (
	"context"
	"strings"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

func TestMoviePlanner_AnimRouting_H4(t *testing.T) {
	tests := []struct {
		name          string
		isAnim        bool
		wantPromptTip string
		wantTargetTip string
	}{
		{
			name:          "anim goes to anim_movie",
			isAnim:        true,
			wantPromptTip: "anim_movie/Chinese",
			wantTargetTip: "anim_movie/",
		},
		{
			name:          "non-anim goes to movie",
			isAnim:        false,
			wantPromptTip: "movie/Chinese",
			wantTargetTip: "movie/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov := mock.NewProvider()
			prov.AddRule(mock.Rule{
				PromptPattern: "organizes movie downloads",
				Response: `{"plan":[{"file":"Movie/Movie.2001.mkv","action":"move",
					"target":"` + tt.wantTargetTip + `Chinese/千与千寻 (2001)/千与千寻 (2001).mkv"}]}`,
			})

			planner := NewMoviePlanner(prov)
			pc := &PlannerContext{
				Files: []string{"Movie/Movie.2001.mkv"},
				Metadata: model.EnrichedMetadata{
					Title:    "千与千寻",
					Year:     2001,
					IsAnim:   tt.isAnim,
					Language: model.LanguageChinese,
				},
			}

			actions, err := planner.Plan(context.Background(), pc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			calls := prov.Calls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 provider call, got %d", len(calls))
			}
			if !strings.Contains(calls[0].Prompt, tt.wantPromptTip) {
				t.Fatalf("prompt root_path mismatch: want tip %q in prompt", tt.wantPromptTip)
			}

			action := findAction(t, actions, "Movie/Movie.2001.mkv")
			if action.Action != "move" {
				t.Fatalf("expected move action, got %s", action.Action)
			}
			if action.Target == nil || !strings.HasPrefix(*action.Target, tt.wantTargetTip) {
				t.Fatalf("expected target starting with %s, got %v", tt.wantTargetTip, action.Target)
			}
		})
	}
}

func TestMoviePlanner_SampleAndTrailerFiltering(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "organizes movie downloads",
		Response: `{"plan":[
			{"file":"Movie/The.Matrix.1999.mkv","action":"move","target":"movie/English/黑客帝国 (1999)/黑客帝国 (1999).mkv"},
			{"file":"Movie/sample.mkv","action":"skip"},
			{"file":"Movie/trailer.mkv","action":"skip"}
		]}`,
	})

	planner := NewMoviePlanner(prov)
	pc := &PlannerContext{
		Files: []string{
			"Movie/The.Matrix.1999.mkv",
			"Movie/sample.mkv",
			"Movie/trailer.mkv",
			"Movie/cover.nfo",
		},
		Metadata: model.EnrichedMetadata{
			Title:    "黑客帝国",
			Year:     1999,
			Language: model.LanguageEnglish,
		},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(actions) != len(pc.Files) {
		t.Fatalf("expected %d actions, got %d", len(pc.Files), len(actions))
	}

	main := findAction(t, actions, "Movie/The.Matrix.1999.mkv")
	if main.Action != "move" || main.Target == nil || *main.Target != "movie/English/黑客帝国 (1999)/黑客帝国 (1999).mkv" {
		t.Fatalf("unexpected main feature action: %s %v", main.Action, main.Target)
	}

	for _, file := range []string{"Movie/sample.mkv", "Movie/trailer.mkv", "Movie/cover.nfo"} {
		action := findAction(t, actions, file)
		if action.Action != "skip" || action.Target != nil {
			t.Fatalf("file %s: want skip with null target, got %s %v", file, action.Action, action.Target)
		}
	}
}
