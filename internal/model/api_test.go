package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlanActionJSON(t *testing.T) {
	// Test move action with target
	targetPath := "movie/Chinese/Test (2024)/Test (2024).mkv"
	actionMove := PlanAction{
		File:   "test.mkv",
		Action: "move",
		Target: &targetPath,
	}

	dataMove, err := json.Marshal(actionMove)
	if err != nil {
		t.Fatalf("failed to marshal move action: %v", err)
	}
	expectedMove := `{"file":"test.mkv","action":"move","target":"movie/Chinese/Test (2024)/Test (2024).mkv"}`
	if string(dataMove) != expectedMove {
		t.Errorf("got %s, want %s", string(dataMove), expectedMove)
	}

	// Test skip action with Target == nil, must explicitly serialize as "target":null (L12)
	actionSkip := PlanAction{
		File:   "sample.mkv",
		Action: "skip",
		Target: nil,
	}

	dataSkip, err := json.Marshal(actionSkip)
	if err != nil {
		t.Fatalf("failed to marshal skip action: %v", err)
	}
	expectedSkip := `{"file":"sample.mkv","action":"skip","target":null}`
	if string(dataSkip) != expectedSkip {
		t.Errorf("got %s, want %s", string(dataSkip), expectedSkip)
	}

	// Round-trip test
	var unmarshaledSkip PlanAction
	if err := json.Unmarshal(dataSkip, &unmarshaledSkip); err != nil {
		t.Fatalf("failed to unmarshal skip action: %v", err)
	}
	if unmarshaledSkip.Target != nil {
		t.Errorf("expected unmarshaled Target to be nil, got %v", *unmarshaledSkip.Target)
	}
	if unmarshaledSkip.File != "sample.mkv" || unmarshaledSkip.Action != "skip" {
		t.Errorf("unexpected unmarshaled fields: %+v", unmarshaledSkip)
	}
}

func TestPlanResponseJSON(t *testing.T) {
	// When Error is nil, it should marshal as "error":null
	resp := PlanResponse{
		Plan:  []PlanAction{},
		Error: nil,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal PlanResponse: %v", err)
	}
	if !strings.Contains(string(data), `"error":null`) {
		t.Errorf("expected PlanResponse to contain '\"error\":null', got %s", string(data))
	}

	// When Error has value
	errMsg := "something failed"
	respWithError := PlanResponse{
		Plan:  []PlanAction{},
		Error: &errMsg,
	}
	dataWithErr, err := json.Marshal(respWithError)
	if err != nil {
		t.Fatalf("failed to marshal PlanResponse with error: %v", err)
	}
	if !strings.Contains(string(dataWithErr), `"error":"something failed"`) {
		t.Errorf("expected PlanResponse to contain error message, got %s", string(dataWithErr))
	}
}

func TestAPIRequestsAndResponses(t *testing.T) {
	// APIPlanRequest round-trip
	planReqJSON := `{"dir":"/incoming/movie","files":["a.mkv","b.nfo"],"metadata":{"dmm_id":"123"}}`
	var planReq APIPlanRequest
	if err := json.Unmarshal([]byte(planReqJSON), &planReq); err != nil {
		t.Fatalf("unmarshal APIPlanRequest failed: %v", err)
	}
	if planReq.Dir != "/incoming/movie" || len(planReq.Files) != 2 || planReq.Metadata["dmm_id"] != "123" {
		t.Errorf("unexpected APIPlanRequest: %+v", planReq)
	}

	// APIExecuteRequest & ExecuteResponse round-trip
	execReqJSON := `{"dir":"/incoming/movie","plan":[{"file":"a.mkv","action":"move","target":"movie/a.mkv"}]}`
	var execReq APIExecuteRequest
	if err := json.Unmarshal([]byte(execReqJSON), &execReq); err != nil {
		t.Fatalf("unmarshal APIExecuteRequest failed: %v", err)
	}
	if len(execReq.Plan) != 1 || *execReq.Plan[0].Target != "movie/a.mkv" {
		t.Errorf("unexpected APIExecuteRequest: %+v", execReq)
	}

	// ExecuteResponse with PlanFailed
	target := "movie/b.mkv"
	execResp := ExecuteResponse{
		FailedMove: []PlanFailed{
			{
				File:   "b.mkv",
				Action: "move",
				Target: &target,
				Reason: "file not found",
			},
		},
	}
	dataResp, err := json.Marshal(execResp)
	if err != nil {
		t.Fatalf("marshal ExecuteResponse failed: %v", err)
	}
	if !strings.Contains(string(dataResp), `"reason":"file not found"`) {
		t.Errorf("ExecuteResponse missing reason: %s", string(dataResp))
	}

	// APIReplanRequest round-trip
	replanReqJSON := `{"files":["a.mkv"],"metadata":null,"previous_response":{"plan":[],"error":null},"user_hint":"this is tv"}`
	var replanReq APIReplanRequest
	if err := json.Unmarshal([]byte(replanReqJSON), &replanReq); err != nil {
		t.Fatalf("unmarshal APIReplanRequest failed: %v", err)
	}
	if replanReq.UserHint != "this is tv" || len(replanReq.Files) != 1 {
		t.Errorf("unexpected APIReplanRequest: %+v", replanReq)
	}
}
