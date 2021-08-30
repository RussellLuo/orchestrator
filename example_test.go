package orchestrator_test

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func Example() {
	o := orchestrator.New()
	builtin.RegisterIn(o)

	task, err := o.Construct(&orchestrator.TaskDefinition{
		Name:    "get_todo_user",
		Type:    builtin.TypeSerial,
		Timeout: 3 * time.Second,
		InputTemplate: orchestrator.InputTemplate{
			"tasks": []*orchestrator.TaskDefinition{
				{
					Name:    "get_todo",
					Type:    builtin.TypeHTTP,
					Timeout: 2 * time.Second,
					InputTemplate: orchestrator.InputTemplate{
						"method": "GET",
						"uri":    "https://jsonplaceholder.typicode.com/todos/${context.input.todoId}",
					},
				},
				{
					Name:    "get_user",
					Type:    builtin.TypeHTTP,
					Timeout: 2 * time.Second,
					InputTemplate: orchestrator.InputTemplate{
						"method": "GET",
						"uri":    "https://jsonplaceholder.typicode.com/users/${get_todo.output.body.userId}",
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	decoder := orchestrator.NewDecoder()
	decoder.AddInput("context", map[string]interface{}{"todoId": 1})
	output, err := task.Execute(context.Background(), decoder)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]interface{})
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}
