package builtin_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

var (
	testOrchestrator = o.New()
)

func TestMain(m *testing.M) {
	builtin.RegisterIn(testOrchestrator)
	os.Exit(m.Run())
}

func TestSerial_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inDef      *o.TaskDefinition
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "ok",
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
			wantOutput: o.Output{"hello": "world"},
		},
		{
			name: "error",
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
									return nil, fmt.Errorf("error in say_name")
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
			wantErr: "error in say_name",
		},
		{
			name: "timeout",
			inDef: &o.TaskDefinition{
				Name:    "greeting",
				Timeout: 50 * time.Millisecond,
				InputTemplate: o.InputTemplate{
					"tasks": []*o.TaskDefinition{
						{
							Name: "say_name",
							Type: builtin.TypeFunc,
							InputTemplate: o.InputTemplate{
								"func": func(context.Context, *o.Decoder) (o.Output, error) {
									time.Sleep(100 * time.Millisecond) // This leads to timeout

									return o.Output{"name": "world"}, nil
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
			wantErr: "context deadline exceeded",
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
