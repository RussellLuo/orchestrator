package orchestrator_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
	"github.com/google/go-cmp/cmp"
)

func TestConstructDecoder(t *testing.T) {
	tests := []struct {
		name          string
		inTaskDef     map[string]any
		wantTaskInput string
	}{
		{
			name: "serial",
			inTaskDef: map[string]any{
				"name":    "greeting",
				"type":    builtin.TypeSerial,
				"timeout": time.Second,
				"input": map[string]any{
					"tasks": []map[string]any{
						{
							"name": "say_name",
							"type": builtin.TypeFunc,
							"input": map[string]any{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
						{
							"name": "say_hello",
							"type": builtin.TypeFunc,
							"input": map[string]any{
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
			inTaskDef: map[string]any{
				"name":    "count",
				"type":    builtin.TypeParallel,
				"timeout": time.Second,
				"input": map[string]any{
					"tasks": []map[string]any{
						{
							"name": "one",
							"type": builtin.TypeFunc,
							"input": map[string]any{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
						{
							"name": "two",
							"type": builtin.TypeFunc,
							"input": map[string]any{
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
			inTaskDef: map[string]any{
				"name":    "count",
				"type":    builtin.TypeHTTP,
				"timeout": time.Second,
				"input": map[string]any{
					"method": "GET",
					"uri":    "https://example.com",
				},
			},
			wantTaskInput: "http(name:count, timeout:1s, request:GET https://example.com, header:<nil>, body:<nil>)",
		},
		{
			name: "decision",
			inTaskDef: map[string]any{
				"name": "test",
				"type": builtin.TypeDecision,
				"input": map[string]any{
					"expression": 0,
					"cases": map[int]map[string]any{
						0: {
							"name": "case_0",
							"type": builtin.TypeFunc,
							"input": map[string]any{
								"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
							},
						},
					},
					"default": map[string]any{
						"name": "default",
						"type": builtin.TypeFunc,
						"input": map[string]any{
							"func": func(context.Context, orchestrator.Input) (orchestrator.Output, error) { return nil, nil },
						},
					},
				},
			},
			wantTaskInput: "decision(name:test, timeout:0s, expression:0, cases:map[0:func(name:case_0)], default:func(name:default))",
		},
		{
			name: "terminate",
			inTaskDef: map[string]any{
				"name":    "count",
				"type":    builtin.TypeTerminate,
				"timeout": time.Second,
				"input": map[string]any{
					"output": map[string]any(nil),
				},
			},
			wantTaskInput: "terminate(name:count, output:map[], error:<nil>)",
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
		"key6": `{"a":"v1","b":"v2","c":"v3"}`,
		"key7": new(orchestrator.Iterator),
	})
	os.Setenv("TEST_NAME", "test_value")

	tests := []struct {
		name    string
		in      any
		wantOut any
	}{
		{
			name:    "is none",
			in:      "${input.key0 == None}", // https://github.com/google/starlark-go/issues/526
			wantOut: true,
		},
		{
			name:    "string",
			in:      "${input.key1}",
			wantOut: "value",
		},
		{
			name:    "empty list",
			in:      "${[]}",
			wantOut: []any{},
		},
		{
			name:    "empty map",
			in:      "${{}}",
			wantOut: map[string]any{},
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
			name:    "get env",
			in:      "${getenv(\"TEST_NAME\")}",
			wantOut: "test_value",
		},
		{
			name:    "json encode",
			in:      "${jsonencode(input.key5)}",
			wantOut: `{"a":"v1","b":"v2","c":"v3"}`,
		},
		{
			name: "json decode",
			in:   `${jsondecode(input.key6)}`,
			wantOut: map[string]any{
				"a": "v1",
				"b": "v2",
				"c": "v3",
			},
		},
		{
			name:    "is iterator",
			in:      "${isiterator(input.key7)}",
			wantOut: true,
		},
		{
			name:    "get iterator",
			in:      "${input.key7}",
			wantOut: new(orchestrator.Iterator),
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
