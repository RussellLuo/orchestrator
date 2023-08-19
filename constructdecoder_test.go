package orchestrator_test

import (
	"context"
	"os"
	"testing"
	"time"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

var (
	testRegistry = o.Registry{}
)

func TestMain(m *testing.M) {
	builtin.MustRegisterDecision(testRegistry)
	builtin.MustRegisterFunc(testRegistry)
	builtin.MustRegisterHTTP(testRegistry)
	builtin.MustRegisterParallel(testRegistry)
	builtin.MustRegisterSerial(testRegistry)
	builtin.MustRegisterTerminate(testRegistry)
	os.Exit(m.Run())
}

func TestConstruct(t *testing.T) {
	tests := []struct {
		name          string
		inTaskDef     *o.TaskDefinition
		wantTaskInput string
	}{
		{
			name: "serial",
			inTaskDef: &o.TaskDefinition{
				Name:    "greeting",
				Type:    builtin.TypeSerial,
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "say_name",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
							},
						},
						{
							Name: "say_hello",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
							},
						},
					},
				},
			},
			wantTaskInput: "serial(name:greeting, timeout:1s, tasks:[func(name:say_name), func(name:say_hello)])",
		},
		{
			name: "parallel",
			inTaskDef: &o.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeParallel,
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "one",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
							},
						},
						{
							Name: "two",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
							},
						},
					},
				},
			},
			wantTaskInput: "parallel(name:count, timeout:1s, tasks:[func(name:one), func(name:two)])",
		},
		{
			name: "http",
			inTaskDef: &o.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeHTTP,
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"method": "GET",
					"uri":    "https://example.com",
				},
			},
			wantTaskInput: "http(name:count, timeout:1s, request:GET https://example.com, header:map[], body:map[])",
		},
		{
			name: "decision",
			inTaskDef: &o.TaskDefinition{
				Name: "test",
				Type: builtin.TypeDecision,
				InputTemplate: o.InputTemplate{
					"switch": 0,
					"cases": map[int]*o.TaskDefinition{
						0: {
							Name: "case_0",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
							},
						},
					},
					"default": &o.TaskDefinition{
						Name: "default",
						Type: builtin.TypeFunc,
						InputTemplate: o.InputTemplate{
							"func": func(context.Context, o.Input) (o.Output, error) { return nil, nil },
						},
					},
				},
			},
			wantTaskInput: "decision(name:test, timeout:0s, switch:0, cases:map[0:func(name:case_0)], default:func(name:default))",
		},
		{
			name: "terminate",
			inTaskDef: &o.TaskDefinition{
				Name:    "count",
				Type:    builtin.TypeTerminate,
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"output": nil,
				},
			},
			wantTaskInput: "terminate(name:count, output:map[])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := testRegistry.Construct(o.NewConstructDecoder(testRegistry), tt.inTaskDef)
			if err != nil {
				t.Fatalf("Err: %v", err)
			}

			if task.InputString() != tt.wantTaskInput {
				t.Fatalf("Task Input: Got (%#v) != Want (%#v)", task.InputString(), tt.wantTaskInput)
			}
		})
	}
}
