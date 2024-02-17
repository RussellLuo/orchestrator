package builtin_test

import (
	"context"
	"fmt"
	"testing"

	o "github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestHTTP_Execute(t *testing.T) {
	tests := []struct {
		name       string
		inInput    map[string]any
		inTask     o.Task
		wantOutput o.Output
		wantErr    string
	}{
		{
			name: "object JSON response",
			inInput: map[string]any{
				"todoId": 1,
			},
			inTask: builtin.NewHTTP("test").Get(
				"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
			),
			wantOutput: o.Output{
				"status": 200,
				"body": map[string]any{
					"userId":    1,
					"id":        1,
					"title":     "delectus aut autem",
					"completed": false,
				},
			},
		},
		{
			name: "array JSON response",
			inInput: map[string]any{
				"todoId": 1,
			},
			inTask: builtin.NewHTTP("test").Get(
				"https://jsonplaceholder.typicode.com/todos?id=${input.todoId}",
			),
			wantOutput: o.Output{
				"status": 200,
				"body": []any{
					map[string]any{
						"userId":    1,
						"id":        1,
						"title":     "delectus aut autem",
						"completed": false,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := o.NewInput(tt.inInput)
			output, err := tt.inTask.Execute(context.Background(), input)

			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if gotErr != tt.wantErr {
				t.Fatalf("Err: Got (%q) != Want (%q)", gotErr, tt.wantErr)
			}

			// Remove the header for simplicity.
			delete(output, "header")

			if fmt.Sprintf("%#v", output) != fmt.Sprintf("%#v", tt.wantOutput) {
				t.Fatalf("Output: Got (%#v) != Want (%#v)", output, tt.wantOutput)
			}
		})
	}
}
