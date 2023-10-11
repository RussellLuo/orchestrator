package builtin_test

import (
	"context"
	"fmt"
	"testing"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestDecision_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inTask     o.Task
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "case hit",
			inTask: builtin.NewDecision("test").
				Expression(0).
				Case(0, builtin.NewFunc("case_0").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "case_0"}, nil
				})).
				Default(builtin.NewFunc("default").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "default"}, nil
				})),
			wantOutput: o.Output{"result": "case_0"},
		},
		{
			name: "default hit",
			inTask: builtin.NewDecision("test").
				Expression(1).
				Case(0, builtin.NewFunc("case_0").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "case_0"}, nil
				})).
				Default(builtin.NewFunc("default").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "default"}, nil
				})),
			wantOutput: o.Output{"result": "default"},
		},
		{
			name: "switch template",
			inTask: builtin.NewDecision("test").
				Expression("${input.value}").
				Case(0, builtin.NewFunc("case_0").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "case_0"}, nil
				})).
				Default(builtin.NewFunc("default").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"result": "default"}, nil
				})),
			wantOutput: o.Output{"result": "case_0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := o.NewInput(map[string]any{"value": 0})
			output, err := tt.inTask.Execute(context.Background(), input)

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
