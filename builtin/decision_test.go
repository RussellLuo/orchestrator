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
		inDef      *o.TaskDefinition
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "case hit",
			inDef: &o.TaskDefinition{
				Name: "test",
				InputTemplate: o.InputTemplate{
					"switch": 0,
					"cases": map[int]*o.TaskDefinition{
						0: {
							Name: "case_0",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "case_0"}, nil
								},
							},
						},
					},
					"default": &o.TaskDefinition{
						Name: "default",
						Type: builtin.TypeFunc,
						InputTemplate: o.InputTemplate{
							"func": func(context.Context, *o.Decoder) (o.Output, error) {
								return o.Output{"result": "default"}, nil
							},
						},
					},
				},
			},
			wantOutput: o.Output{"result": "case_0"},
		},
		{
			name: "default hit",
			inDef: &o.TaskDefinition{
				Name: "test",
				InputTemplate: o.InputTemplate{
					"switch": 1,
					"cases": map[int]*o.TaskDefinition{
						0: {
							Name: "case_0",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "case_0"}, nil
								},
							},
						},
					},
					"default": &o.TaskDefinition{
						Name: "default",
						Type: builtin.TypeFunc,
						InputTemplate: o.InputTemplate{
							"func": func(context.Context, *o.Decoder) (o.Output, error) {
								return o.Output{"result": "default"}, nil
							},
						},
					},
				},
			},
			wantOutput: o.Output{"result": "default"},
		},
		{
			name: "switch template",
			inDef: &o.TaskDefinition{
				Name: "test",
				InputTemplate: o.InputTemplate{
					"switch": "${context.input.value}",
					"cases": map[int]*o.TaskDefinition{
						0: {
							Name: "case_0",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"result": "case_0"}, nil
								},
							},
						},
					},
					"default": &o.TaskDefinition{
						Name: "default",
						Type: builtin.TypeFunc,
						InputTemplate: o.InputTemplate{
							"func": func(context.Context, *o.Decoder) (o.Output, error) {
								return o.Output{"result": "default"}, nil
							},
						},
					},
				},
			},
			wantOutput: o.Output{"result": "case_0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := builtin.NewDecision(testOrchestrator, tt.inDef)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			decoder := o.NewDecoder()
			decoder.AddInput("context", map[string]interface{}{"value": 0})
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
