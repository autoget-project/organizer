package mock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"organizer/internal/ai/mock"
)

type MockTarget struct {
	Result string `json:"result"`
	Code   int    `json:"code"`
}

func TestMockProvider_ExactAndRegexRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prompt    string
		want      MockTarget
		wantCalls int
	}{
		{"exact match wins", "exact match", MockTarget{Result: "exact_success", Code: 200}, 1},
		{"regex rule", "classify file: sample.mkv", MockTarget{Result: "regex_matched", Code: 100}, 1},
		{"fallback response", "something else", MockTarget{Result: "fallback", Code: 0}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

			var res MockTarget
			require.NoError(t, p.GenerateStructured(context.Background(), tt.prompt, MockTarget{}, &res))
			assert.Equal(t, tt.want, res)
			assert.Len(t, p.Calls(), tt.wantCalls, "call tracking must record every GenerateStructured call")
		})
	}
}

func TestMockProvider_ErrorRule(t *testing.T) {
	t.Parallel()

	p := mock.NewProvider()
	p.AddRule(mock.Rule{
		PromptPattern: "error prompt",
		Error:         errors.New("simulated failure"),
	})

	var res MockTarget
	err := p.GenerateStructured(context.Background(), "error prompt", MockTarget{}, &res)
	require.Error(t, err)
	assert.EqualError(t, err, "simulated failure")
}
