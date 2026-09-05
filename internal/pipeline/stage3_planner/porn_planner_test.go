package stage3planner

import (
	"context"
	"testing"

	"organizer/internal/model"
)

func TestPornPlanner_NamingFallbackChain_L9(t *testing.T) {
	tests := []struct {
		name string
		pc   *PlannerContext
		want string
	}{
		{
			name: "entities id first",
			pc: &PlannerContext{
				Files:    []string{"Long-Con-1.mp4"},
				Metadata: model.EnrichedMetadata{Title: "Long Con Part 1"},
				Entities: map[string]interface{}{
					"id":   "vixen-long-con-part-1",
					"name": "Long Con Part 1",
				},
			},
			want: "porn/vixen-long-con-part-1/vixen-long-con-part-1.mp4",
		},
		{
			name: "entities name second",
			pc: &PlannerContext{
				Files:    []string{"Long-Con-1.mp4"},
				Metadata: model.EnrichedMetadata{Title: "Should Not Be Used"},
				Entities: map[string]interface{}{
					"name": "Long Con Part 1",
				},
			},
			want: "porn/Long Con Part 1/Long Con Part 1.mp4",
		},
		{
			name: "enriched title third",
			pc: &PlannerContext{
				Files:    []string{"Long-Con-1.mp4"},
				Metadata: model.EnrichedMetadata{Title: "Long Con Part 1"},
			},
			want: "porn/Long Con Part 1/Long Con Part 1.mp4",
		},
		{
			name: "file stem last",
			pc: &PlannerContext{
				Files: []string{"Long-Con-1.mp4"},
			},
			want: "porn/Long-Con-1/Long-Con-1.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner := NewPornPlanner()
			actions, err := planner.Plan(context.Background(), tt.pc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			action := findAction(t, actions, tt.pc.Files[0])
			if action.Action != "move" || action.Target == nil || *action.Target != tt.want {
				t.Fatalf("want move to %s, got %s %v", tt.want, action.Action, action.Target)
			}
		})
	}
}

func TestPornPlanner_VRRootAndCompanionFiles(t *testing.T) {
	planner := NewPornPlanner()
	pc := &PlannerContext{
		Files: []string{
			"VR Video.mp4",
			"VR Video.srt",
			"cover.jpg",
			"meta.nfo",
		},
		Metadata: model.EnrichedMetadata{IsVR: true},
	}

	actions, err := planner.Plan(context.Background(), pc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	move := findAction(t, actions, "VR Video.mp4")
	if move.Action != "move" || move.Target == nil || *move.Target != "porn_vr/VR Video/VR Video.mp4" {
		t.Fatalf("unexpected VR action: %+v", move)
	}

	// Subtitle files stay unplanned (Stage 4 pairs them semantically).
	for _, a := range actions {
		if a.File == "VR Video.srt" {
			t.Fatalf("subtitle must be left to stage 4, got action %+v", a)
		}
	}

	// Non-media files are skipped.
	for _, file := range []string{"cover.jpg", "meta.nfo"} {
		action := findAction(t, actions, file)
		if action.Action != "skip" || action.Target != nil {
			t.Fatalf("file %s: want skip, got %+v", file, action)
		}
	}
}
