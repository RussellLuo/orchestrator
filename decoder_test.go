package orchestrator_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestConstructDecoder(t *testing.T) {
	tests := []struct {
		name          string
		inTaskDef     *orchestrator.TaskDefinition
		wantTaskInput string
	}{
		{
			name: "serial",
			inTaskDef: &orchestrator.TaskDefinition{
				Name:    "greeting",
				Type:    builtin.TypeSerial,
				Timeout: time.Second,
				InputTemplate: orchestrator.InputTemplate{
					"tasks": []*orchestrator.TaskDefinition{
						{
							Name: "say_name",
							Type: builtin.TypeFunc,
							InputTemplate: orchestrator.InputTemplate{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
						{
							Name: "say_hello",
							Type: builtin.TypeFunc,
							InputTemplate: orchestrator.InputTemplate{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
					},
				},
			},
			wantTaskInput: "serial(name:greeting, timeout:1s, tasks:[func(name:say_name), func(name:say_hello)])",
		},
		{
			name: "parallel",
			inTaskDef: &orchestrator.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeParallel,
				Timeout: time.Second,
				InputTemplate: orchestrator.InputTemplate{
					"tasks": []*orchestrator.TaskDefinition{
						{
							Name: "one",
							Type: builtin.TypeFunc,
							InputTemplate: orchestrator.InputTemplate{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
						{
							Name: "two",
							Type: builtin.TypeFunc,
							InputTemplate: orchestrator.InputTemplate{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
					},
				},
			},
			wantTaskInput: "parallel(name:count, timeout:1s, tasks:[func(name:one), func(name:two)])",
		},
		{
			name: "http",
			inTaskDef: &orchestrator.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeHTTP,
				Timeout: time.Second,
				InputTemplate: orchestrator.InputTemplate{
					"method": "GET",
					"uri":    "https://example.com",
				},
			},
			wantTaskInput: "http(name:count, timeout:1s, request:GET https://example.com, header:<nil>, body:<nil>)",
		},
		{
			name: "decision",
			inTaskDef: &orchestrator.TaskDefinition{
				Name: "test",
				Type: builtin.TypeDecision,
				InputTemplate: orchestrator.InputTemplate{
					"expression": 0,
					"cases": map[int]*orchestrator.TaskDefinition{
						0: {
							Name: "case_0",
							Type: builtin.TypeFunc,
							InputTemplate: orchestrator.InputTemplate{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
					},
					"default": &orchestrator.TaskDefinition{
						Name: "default",
						Type: builtin.TypeFunc,
						InputTemplate: orchestrator.InputTemplate{
							"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
						},
					},
				},
			},
			wantTaskInput: "decision(name:test, timeout:0s, expression:0, cases:map[0:func(name:case_0)], default:func(name:default))",
		},
		{
			name: "terminate",
			inTaskDef: &orchestrator.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeTerminate,
				Timeout: time.Second,
				InputTemplate: orchestrator.InputTemplate{
					"output": map[string]any(nil),
				},
			},
			wantTaskInput: "terminate(name:count, output:map[])",
		},
	}

	r := orchestrator.Registry{}
	builtin.MustRegisterDecision(r)
	builtin.MustRegisterFunc(r)
	builtin.MustRegisterHTTP(r)
	builtin.MustRegisterParallel(r)
	builtin.MustRegisterSerial(r)
	builtin.MustRegisterTerminate(r)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := r.Construct(orchestrator.NewConstructDecoder(r), tt.inTaskDef)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			if task.String() != tt.wantTaskInput {
				t.Fatalf("Task Input: Got (%#v) != Want (%#v)", task.String(), tt.wantTaskInput)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	input := orchestrator.NewInput(map[string]any{
		"key1": "value",
		"key2": 0,
		"key3": true,
	})

	tests := []struct {
		name    string
		in      any
		wantOut any
	}{
		{
			name:    "string",
			in:      "${input.key1}",
			wantOut: "value",
		},
		{
			name:    "array",
			in:      []string{"${input.key1}"},
			wantOut: []string{"value"},
		},
		{
			name:    "map",
			in:      map[string]any{"key": "${input.key2}"},
			wantOut: map[string]any{"key": 0},
		},
		{
			name:    "nested map",
			in:      map[string]any{"outer": map[string]any{"inner": "${input.key3}"}},
			wantOut: map[string]any{"outer": map[string]any{"inner": true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := orchestrator.Evaluate(tt.in, input.Evaluate)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}
			if !reflect.DeepEqual(out, tt.wantOut) {
				t.Fatalf("Out: Got (%#v) != Want (%#v)", out, tt.wantOut)
			}
		})
	}
}
