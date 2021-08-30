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
		inDef      *o.TaskDefinition
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "terminate",
			inDef: &o.TaskDefinition{
				Name:    "greeting",
				Timeout: time.Second,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "say_name",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									return o.Output{"name": "world"}, nil
								},
							},
						},
						{
							Name: "say_goodbye",
							Type: builtin.TypeTerminate,
							InputTemplate: o.InputTemplate{
								"output": o.Output{
									"goodbye": "${say_name.output.name}",
								},
							},
						},
						{
							Name: "say_hello",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(ctx context.Context, decoder *o.Decoder) (o.Output, error) {
									input := map[string]interface{}{
										"hello": "${say_name.output.name}",
									}
									output := make(map[string]interface{})
									if err := decoder.Decode(input, &output); err != nil {
										return nil, err
									}
									return output, nil
								},
							},
						},
					},
				},
			},
			wantOutput: o.Output{
				"terminated": true,
				"goodbye":    "world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := builtin.NewSerial(testOrchestrator, tt.inDef)
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
