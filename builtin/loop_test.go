package builtin_test

import (
	"context"
	"fmt"
	"testing"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestLoop_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inInput    map[string]any
		inTask     o.Task
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "loop in",
			inInput: map[string]any{
				"array": []any{0, 1, 2},
			},
			inTask: builtin.NewLoop("test").
				Iterator(builtin.NewIterate("iterator").In("${context.input.array}")).
				Body(builtin.NewFunc("body").Func(func(_ context.Context, input o.Input) (o.Output, error) {
					value := o.Expr[any]{Expr: "${iterator.output.value}"}
					if err := value.Evaluate(input); err != nil {
						return nil, err
					}
					return o.Output{"value": value.Value}, nil
				})),
			wantOutput: o.Output{
				"iteration": 3,
				"0":         map[string]any{"value": 0},
				"1":         map[string]any{"value": 1},
				"2":         map[string]any{"value": 2},
			},
		},
		{
			name: "loop range",
			inInput: map[string]any{
				"start": 3,
				"stop":  6,
			},
			inTask: builtin.NewLoop("test").
				Iterator(builtin.NewIterate("iterator").Range([]any{"${context.input.start}", "${context.input.stop}"})).
				Body(builtin.NewFunc("body").Func(func(_ context.Context, input o.Input) (o.Output, error) {
					value := o.Expr[any]{Expr: "${iterator.output.value}"}
					if err := value.Evaluate(input); err != nil {
						return nil, err
					}
					return o.Output{"value": value.Value}, nil
				})),
			wantOutput: o.Output{
				"iteration": 3,
				"0":         map[string]any{"value": 3},
				"1":         map[string]any{"value": 4},
				"2":         map[string]any{"value": 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := o.NewInput(tt.inInput)
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
