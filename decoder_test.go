package orchestrator_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestDecoder_Decode(t *testing.T) {
	type Out struct {
		A string        `orchestrator:"a"`
		B int           `orchestrator:"b"`
		C string        `orchestrator:"c"`
		D string        `orchestrator:"d"`
		E string        `orchestrator:"e"`
		F time.Duration `orchestrator:"f"`
	}

	decoder := orchestrator.NewDecoder()
	decoder.AddInput("task1", map[string]interface{}{
		"value": "1",
	})
	decoder.AddOutput("task2", map[string]interface{}{
		"value": 2,
	})

	in := map[string]interface{}{
		"a": "${task1.input.value}",
		"b": "${task2.output.value}",
		"c": "/posts/${task2.output.value}",
		"d": "${task1.input.value}.${task2.output.value}",
		"e": "task.output.value",
		"f": "3s",
	}
	var out Out
	if err := decoder.Decode(in, &out); err != nil {
		t.Fatalf("Err: %v", err)
	}

	wantOut := Out{
		A: "1",
		B: 2,
		C: "/posts/2",
		D: "1.2",
		E: "task.output.value",
		F: 3 * time.Second,
	}
	if !reflect.DeepEqual(out, wantOut) {
		t.Fatalf("Out: Got (%#v) != Want (%#v)", out, wantOut)
	}
}

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
			wantTaskInput: "http(name:count, timeout:1s, request:GET https://example.com, header:map[], body:map[])",
		},
		{
			name: "decision",
			inTaskDef: &orchestrator.TaskDefinition{
				Name: "test",
				Type: builtin.TypeDecision,
				InputTemplate: orchestrator.InputTemplate{
					"switch": 0,
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
			wantTaskInput: "decision(name:test, timeout:0s, switch:0, cases:map[0:func(name:case_0)], default:func(name:default))",
		},
		{
			name: "terminate",
			inTaskDef: &orchestrator.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeTerminate,
				Timeout: time.Second,
				InputTemplate: orchestrator.InputTemplate{
					"output": nil,
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

			if task.InputString() != tt.wantTaskInput {
				t.Fatalf("Task Input: Got (%#v) != Want (%#v)", task.InputString(), tt.wantTaskInput)
			}
		})
	}
}
