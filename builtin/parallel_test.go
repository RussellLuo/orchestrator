package builtin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestParallel_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inTask     o.Task
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "ok",
			inTask: builtin.NewParallel("count").Timeout(time.Second).Tasks(
				builtin.NewFunc("one").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number one"}, nil
				}),
				builtin.NewFunc("two").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number two"}, nil
				}),
				builtin.NewFunc("three").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number three"}, nil
				}),
			),
			wantOutput: o.Output{
				"one": o.Output{
					"result": "number one",
				},
				"two": o.Output{
					"result": "number two",
				},
				"three": o.Output{
					"result": "number three",
				},
			},
		},
		{
			name: "error",
			inTask: builtin.NewParallel("count").Timeout(time.Second).Tasks(
				builtin.NewFunc("one").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return nil, fmt.Errorf("the first error")
				}),
				builtin.NewFunc("two").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number two"}, nil
				}),
				builtin.NewFunc("three").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return nil, fmt.Errorf("the third error")
				}),
			),
			wantErr: "the first error; the third error",
		},
		{
			name: "timeout",
			inTask: builtin.NewParallel("count").Timeout(50*time.Millisecond).Tasks(
				builtin.NewFunc("one").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number one"}, nil
				}),
				builtin.NewFunc("two").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					return o.Output{"result": "number two"}, nil
				}),
				builtin.NewFunc("three").Func(func(context.Context, *o.Decoder) (o.Output, error) {
					time.Sleep(100 * time.Millisecond) // This leads to timeout

					return o.Output{"result": "number three"}, nil
				}),
			),
			wantErr: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := o.NewDecoder()
			output, err := tt.inTask.Execute(context.Background(), decoder)

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
