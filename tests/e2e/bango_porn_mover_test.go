package e2e

import (
	"testing"

	"github.com/autoget-project/organizer/internal/ai/mock"
	"github.com/autoget-project/organizer/internal/model"
)

// TestE2E_BangoMoverWithActor drives the bango mover for a release whose
// actress is unknown to the local actor file: the plan creates the actress
// directory by name.
func TestE2E_BangoMoverWithActor(t *testing.T) {
	t.Parallel()

	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"bango_porn","reason":"standard bango","entities":{"bango":"SSIS-698","actors":["Yua Mikami"]}}`})
	prov.AddRule(mock.Rule{PromptPattern: patBango, Response: `{"filenames":[{"file":"SSIS-698-C.mp4","new_filename":"SSIS-698-C.mp4"}]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "downloads",
		Files: []string{"SSIS-698-C.mp4"},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"SSIS-698-C.mp4": wantMove("jav/Yua Mikami/SSIS-698-C.mp4"),
	})
}

// TestE2E_BangoPartMatrixAndCPriority covers the bango derivation matrix and
// priority (M3): ssis-698-a -> part.1, ssis-698-b -> part.2, and ssis-698-C
// keeps its name because the -C Chinese-subtitle rule outranks multi-part
// renumbering.
func TestE2E_BangoPartMatrixAndCPriority(t *testing.T) {
	t.Parallel()

	s, prov := newMockSandbox(t)

	prov.AddRule(mock.Rule{PromptPattern: patClassifier, Response: `{"category":"bango_porn","reason":"bango series","entities":{"bango":"SSIS-698"}}`})
	prov.AddRule(mock.Rule{PromptPattern: patBango, Response: `{"filenames":[
		{"file":"ssis-698-a.mp4","new_filename":"SSIS-698.part.1.mp4"},
		{"file":"ssis-698-b.mp4","new_filename":"SSIS-698.part.2.mp4"},
		{"file":"ssis-698-C.mp4","new_filename":"SSIS-698-C.mp4"}
	]}`})

	code, body := s.postJSON(t, "/v1/plan", model.APIPlanRequest{
		Dir:   "javdl",
		Files: []string{"ssis-698-a.mp4", "ssis-698-b.mp4", "ssis-698-C.mp4"},
	})

	assertPlanContract(t, code, body, map[string]model.PlanAction{
		"ssis-698-a.mp4": wantMove("jav/素人/SSIS-698.part.1.mp4"),
		"ssis-698-b.mp4": wantMove("jav/素人/SSIS-698.part.2.mp4"),
		"ssis-698-C.mp4": wantMove("jav/素人/SSIS-698-C.mp4"),
	})
}
