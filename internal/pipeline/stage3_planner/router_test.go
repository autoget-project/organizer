package stage3planner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

func TestRouter_UnknownCategoryReturnsEmptyPlan(t *testing.T) {
	// No rules and no default response: any LLM call would fail loudly.
	router := NewRouter(mock.NewProvider())

	actions, err := router.Plan(context.Background(), model.CategoryUnknown, &PlannerContext{
		Files: []string{"hash/junk.bin"},
	})
	if err != nil {
		t.Fatalf("unknown category must not produce an error, got %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("unknown category must return an empty plan, got %+v", actions)
	}
}

func TestRouter_SimpleCategoriesRouteToLocalPlanner(t *testing.T) {
	// No rules: if the router wrongly delegated to an LLM planner it would error.
	router := NewRouter(mock.NewProvider())

	for _, cat := range model.SimpleMoveCategories {
		actions, err := router.Plan(context.Background(), cat, &PlannerContext{
			Files: []string{"hash/" + string(cat) + ".dat"},
		})
		if err != nil {
			t.Fatalf("category %s: unexpected error %v", cat, err)
		}
		if len(actions) != 1 || actions[0].Action != "move" {
			t.Fatalf("category %s: unexpected plan %+v", cat, actions)
		}
		wantPrefix := string(cat) + "/"
		if actions[0].Target == nil || !strings.HasPrefix(*actions[0].Target, wantPrefix) {
			t.Fatalf("category %s: expected target prefix %s, got %v", cat, wantPrefix, actions[0].Target)
		}
	}
}

func TestRouter_PornRoutesToLocalPlanner(t *testing.T) {
	router := NewRouter(mock.NewProvider())

	actions, err := router.Plan(context.Background(), model.CategoryPorn, &PlannerContext{
		Files: []string{"Long-Con-1.mp4"},
		Entities: map[string]interface{}{
			"id": "vixen-long-con-part-1",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	action := findAction(t, actions, "Long-Con-1.mp4")
	if action.Target == nil || *action.Target != "porn/vixen-long-con-part-1/vixen-long-con-part-1.mp4" {
		t.Fatalf("unexpected porn plan: %+v", action)
	}
}

func TestRouter_LLMPlannersRouteToProvider(t *testing.T) {
	// A failing default response proves which planner each category reaches.
	prov := mock.NewProvider()
	prov.SetDefaultResponse(nil, fmt.Errorf("no rule"))
	router := NewRouter(prov)

	tests := []struct {
		cat       model.Category
		errSubstr string
	}{
		{model.CategoryTVSeries, "tv planner llm"},
		{model.CategoryMovie, "movie planner llm"},
		{model.CategoryBangoPorn, "bango planner llm"},
	}

	for _, tt := range tests {
		_, err := router.Plan(context.Background(), tt.cat, &PlannerContext{
			Files:    []string{"X/E01.mkv"},
			Metadata: model.EnrichedMetadata{Title: "X"},
		})
		if err == nil {
			t.Fatalf("category %s: expected provider error propagation", tt.cat)
		}
		if !strings.Contains(err.Error(), tt.errSubstr) {
			t.Fatalf("category %s: error %q does not mention %q", tt.cat, err.Error(), tt.errSubstr)
		}
	}
}
