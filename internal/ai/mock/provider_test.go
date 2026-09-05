package mock_test

import (
	"context"
	"fmt"
	"testing"

	"organizer/internal/ai/mock"
)

type MockTarget struct {
	Result string `json:"result"`
	Code   int    `json:"code"`
}

func TestMockProvider_ExactAndRegexRules(t *testing.T) {
	p := mock.NewProvider()

	p.AddRule(mock.Rule{
		PromptPattern: "exact match",
		Response:      MockTarget{Result: "exact_success", Code: 200},
	})

	p.AddRule(mock.Rule{
		PromptPattern: `classify\s+file:\s+(\w+\.mkv)`,
		IsRegex:       true,
		Response:      `{"result":"regex_matched","code":100}`,
	})

	p.SetDefaultResponse(MockTarget{Result: "fallback", Code: 0}, nil)

	// Test 1: exact match
	var res1 MockTarget
	if err := p.GenerateStructured(context.Background(), "exact match", MockTarget{}, &res1); err != nil {
		t.Fatalf("res1 failed: %v", err)
	}
	if res1.Result != "exact_success" || res1.Code != 200 {
		t.Errorf("unexpected res1: %+v", res1)
	}

	// Test 2: regex match
	var res2 MockTarget
	if err := p.GenerateStructured(context.Background(), "classify file: sample.mkv", MockTarget{}, &res2); err != nil {
		t.Fatalf("res2 failed: %v", err)
	}
	if res2.Result != "regex_matched" || res2.Code != 100 {
		t.Errorf("unexpected res2: %+v", res2)
	}

	// Test 3: fallback
	var res3 MockTarget
	if err := p.GenerateStructured(context.Background(), "something else", MockTarget{}, &res3); err != nil {
		t.Fatalf("res3 failed: %v", err)
	}
	if res3.Result != "fallback" || res3.Code != 0 {
		t.Errorf("unexpected res3: %+v", res3)
	}

	// Verify call tracking
	calls := p.Calls()
	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
	}
}

func TestMockProvider_ErrorRule(t *testing.T) {
	p := mock.NewProvider()
	p.AddRule(mock.Rule{
		PromptPattern: "error prompt",
		Error:         fmt.Errorf("simulated failure"),
	})

	var res MockTarget
	err := p.GenerateStructured(context.Background(), "error prompt", MockTarget{}, &res)
	if err == nil || err.Error() != "simulated failure" {
		t.Fatalf("expected simulated failure, got %v", err)
	}
}
