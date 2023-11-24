package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
	"github.com/google/go-cmp/cmp"
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

	r := orchestrator.NewRegistry()
	builtin.MustRegisterDecision(r)
	builtin.MustRegisterFunc(r)
	builtin.MustRegisterHTTP(r)
	builtin.MustRegisterParallel(r)
	builtin.MustRegisterSerial(r)
	builtin.MustRegisterTerminate(r)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := r.Construct(tt.inTaskDef)
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
		"key4": []any{
			1,
			2,
			3,
		},
		"key5": map[string]any{
			"a": "v1",
			"b": "v2",
			"c": "v3",
		},
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
			name: "list comprehension",
			in:   "${[x*2 for x in input.key4]}",
			wantOut: []any{
				2,
				4,
				6,
			},
		},
		{
			name: "dictionary comprehension",
			in:   "${{k: v.upper() for k, v in input.key5.items()}}",
			wantOut: map[string]any{
				"a": "V1",
				"b": "V2",
				"c": "V3",
			},
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
			got, err := orchestrator.Evaluate(tt.in, input.Evaluate)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}
			if !cmp.Equal(got, tt.wantOut) {
				diff := cmp.Diff(got, tt.wantOut)
				t.Errorf("Want - Got: %s", diff)
			}
		})
	}
}
