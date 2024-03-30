package builtin_test

import (
	"context"
	"fmt"
	"testing"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestCode_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inCode     o.Task
		inInput    map[string]any
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "list comprehension",
			inCode: builtin.NewCode("test").
				Code(`
def _(env):
    return [x*2 for x in env.input.values]
`).Build(),
			inInput: map[string]any{
				"values": []any{1, 2, 3},
			},
			wantOutput: o.Output{"result": []any{2, 4, 6}},
		},
		{
			name: "add fields",
			inCode: builtin.NewCode("test").
				Code(`
def _(env):
    for x in env.input.values:
        x.k2 = "v2"
    return env.input.values
`).Build(),
			inInput: map[string]any{
				"values": []any{
					map[string]any{
						"k1": "v1",
					},
					map[string]any{
						"k1": "v1",
					},
					map[string]any{
						"k1": "v1",
					},
				},
			},
			wantOutput: o.Output{
				"result": []any{
					map[string]any{
						"k1": "v1",
						"k2": "v2",
					},
					map[string]any{
						"k1": "v1",
						"k2": "v2",
					},
					map[string]any{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := o.NewInput(tt.inInput)
			output, err := tt.inCode.Execute(context.Background(), input)

			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if gotErr != tt.wantErr {
				t.Fatalf("Err: Got (%q) != Want (%q)", gotErr, tt.wantErr)
			}

			if fmt.Sprintf("%#v", output) != fmt.Sprintf("%#v", tt.wantOutput) {
				t.Fatalf("Output: Got (%#v) != Want (%#v)", output, tt.wantOutput)
			}
		})
	}
}
