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
		inDef      *o.TaskDefinition
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "ok",
			inDef: &o.TaskDefinition{
				Name:    "count",
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "one",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "number one"}, nil
								},
							},
						},
						{
							Name: "two",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "number two"}, nil
								},
							},
						},
						{
							Name: "three",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "number three"}, nil
								},
							},
						},
					},
				},
			},
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
			inDef: &o.TaskDefinition{
				Name:    "count",
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "one",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return nil, fmt.Errorf("the first error")
								},
							},
						},
						{
							Name: "two",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "number two"}, nil
								},
							},
						},
						{
							Name: "three",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return nil, fmt.Errorf("the third error")
								},
							},
						},
					},
				},
			},
			wantErr: "the first error; the third error",
		},
		{
			name: "timeout",
			inDef: &o.TaskDefinition{
				Name:    "count",
				Timeout: 50 * time.Millisecond,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "one",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return nil, fmt.Errorf("the first error")
								},
							},
						},
						{
							Name: "two",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "number two"}, nil
								},
							},
						},
						{
							Name: "three",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									time.Sleep(100 * time.Millisecond) // This leads to timeout

									return nil, fmt.Errorf("the third error")
								},
							},
						},
					},
				},
			},
			wantErr: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := builtin.NewParallel(testOrchestrator, tt.inDef)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			decoder := o.NewDecoder()
			output, err := task.Execute(context.Background(), decoder)

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
