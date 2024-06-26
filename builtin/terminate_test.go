package builtin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestTerminate(t *testing.T) {
	tests := []struct {
		name       string
		inTask     o.Task
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "terminate",
			inTask: builtin.NewSerial("greeting").Timeout(time.Second).Tasks(
				builtin.NewFunc("say_name").Func(func(context.Context, o.Input) (o.Output, error) {
					return o.Output{"name": "world"}, nil
				}),
				builtin.NewTerminate("say_goodbye").Output(o.Output{
					"goodbye": "${say_name.name}",
				}),
				builtin.NewFunc("say_hello").Func(func(ctx context.Context, input o.Input) (o.Output, error) {
					in := o.Expr[map[string]any]{
						Expr: map[string]any{
							"hello": "${say_name.name}",
						},
					}
					if err := in.Evaluate(input); err != nil {
						return nil, err
					}
					return in.Value, nil
				}),
			).Build(),
			wantOutput: o.Output{
				"terminated": true,
				"goodbye":    "world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := o.NewInput(nil)
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
