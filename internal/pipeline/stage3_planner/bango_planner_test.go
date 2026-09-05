package stage3planner

import (
	"context"
	"testing"

	"organizer/internal/ai/mock"
	"organizer/internal/model"
)

func TestResolveBangoTargetDir_DecisionMatrix(t *testing.T) {
	tests := []struct {
		name string
		meta model.EnrichedMetadata
		want string
	}{
		{
			name: "default jav with amateur dir",
			meta: model.EnrichedMetadata{Bango: "SSIS-698"},
			want: "jav/素人",
		},
		{
			name: "jav_vr root with bango subdir",
			meta: model.EnrichedMetadata{Bango: "SSIS-698", IsVR: true},
			want: "jav_vr/素人/SSIS-698",
		},
		{
			name: "madou root has top priority",
			meta: model.EnrichedMetadata{Bango: "MD-0123", FromMadou: true},
			want: "madou/素人",
		},
		{
			name: "resolved actor directory",
			meta: model.EnrichedMetadata{Bango: "SSIS-698", Actors: []string{"Yui Hatano", "波多野結衣"}},
			want: "jav/Yui Hatano",
		},
		{
			name: "vr with actor appends bango subdir",
			meta: model.EnrichedMetadata{Bango: "SSIS-698", IsVR: true, Actors: []string{"Yui Hatano"}},
			want: "jav_vr/Yui Hatano/SSIS-698",
		},
		{
			name: "madou overrides vr root but keeps bango subdir",
			meta: model.EnrichedMetadata{Bango: "MD-0123", IsVR: true, FromMadou: true},
			want: "madou/素人/MD-0123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveBangoTargetDir(tt.meta); got != tt.want {
				t.Fatalf("ResolveBangoTargetDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBangoPlanner_CPriorityOverMultiPart(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "bango porn videos",
		Response: `{"filenames":[
			{"file":"/downloads/SSIS-698-C.mp4","new_filename":"SSIS-698-C.mp4"},
			{"file":"/downloads/SSIS-698-A.mp4","new_filename":"SSIS-698.part.1.mp4"},
			{"file":"/downloads/SSIS-698-B.mp4","new_filename":"SSIS-698.part.2.mp4"}
		]}`,
	})

	planner := NewBangoPlanner(prov)
	pc := &PlannerContext{
		Files: []string{
			"/downloads/SSIS-698-C.mp4",
			"/downloads/SSIS-698-A.mp4",
			"/downloads/SSIS-698-B.mp4",
		},
		Metadata: model.EnrichedMetadata{
			Bango:  "SSIS-698",
			Actors: []string{"Yua Mikami"},
		},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"/downloads/SSIS-698-C.mp4": "jav/Yua Mikami/SSIS-698-C.mp4",
		"/downloads/SSIS-698-A.mp4": "jav/Yua Mikami/SSIS-698.part.1.mp4",
		"/downloads/SSIS-698-B.mp4": "jav/Yua Mikami/SSIS-698.part.2.mp4",
	}
	for file, target := range want {
		action := findAction(t, actions, file)
		if action.Action != "move" || action.Target == nil || *action.Target != target {
			t.Fatalf("file %s: want move to %s, got %s %v", file, target, action.Action, action.Target)
		}
	}
}

func TestBangoPlanner_PortedeJAVActorCase(t *testing.T) {
	// Ported from archived/app/agents/mover/bango_porn_mover_test.py:
	// single -C file with a known actor dir resolved by Stage 2.
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "bango porn videos",
		Response:      `{"filenames":[{"file":"/downloads/SSIS-698-C.mp4","new_filename":"SSIS-698-C.mp4"}]}`,
	})

	planner := NewBangoPlanner(prov)
	pc := &PlannerContext{
		Files:    []string{"/downloads/SSIS-698-C.mp4"},
		Metadata: model.EnrichedMetadata{Bango: "SSIS-698", Actors: []string{"Yua Mikami"}},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	action := findAction(t, actions, "/downloads/SSIS-698-C.mp4")
	if action.Action != "move" || action.Target == nil || *action.Target != "jav/Yua Mikami/SSIS-698-C.mp4" {
		t.Fatalf("unexpected action: %+v", action)
	}
}

func TestBangoPlanner_MadouAndNonVideoSkip(t *testing.T) {
	prov := mock.NewProvider()
	prov.AddRule(mock.Rule{
		PromptPattern: "bango porn videos",
		Response:      `{"filenames":[{"file":"/downloads/MD-0123.mp4","new_filename":"MD-0123.mp4"}]}`,
	})

	planner := NewBangoPlanner(prov)
	pc := &PlannerContext{
		Files:    []string{"/downloads/MD-0123.mp4", "/downloads/cover.jpg"},
		Metadata: model.EnrichedMetadata{Bango: "MD-0123", FromMadou: true},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	move := findAction(t, actions, "/downloads/MD-0123.mp4")
	if move.Action != "move" || move.Target == nil || *move.Target != "madou/素人/MD-0123.mp4" {
		t.Fatalf("unexpected madou action: %+v", move)
	}

	skip := findAction(t, actions, "/downloads/cover.jpg")
	if skip.Action != "skip" || skip.Target != nil {
		t.Fatalf("unexpected non-video action: %+v", skip)
	}
}
