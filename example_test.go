package orchestrator_test

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func Example() {
	task := builtin.NewSerial("get_todo_user").Timeout(3*time.Second).Tasks(
		builtin.NewHTTP("get_todo").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/todos/${context.input.todoId}",
		),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.output.body.userId}",
		),
	)

	input := orchestrator.NewInput(map[string]interface{}{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]interface{})
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham

}

func Example_construct() {
	r := orchestrator.Registry{}
	builtin.MustRegisterSerial(r)
	builtin.MustRegisterHTTP(r)

	task, err := r.Construct(orchestrator.NewConstructDecoder(r), &orchestrator.TaskDefinition{
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

	input := orchestrator.NewInput(map[string]interface{}{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]interface{})
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}

func Example_constructFromJSON() {
	r := orchestrator.Registry{}
	builtin.MustRegisterSerial(r)
	builtin.MustRegisterHTTP(r)

	data := []byte(`{
  "name": "get_todo_user",
  "type": "serial",
  "timeout": "3s",
  "input": {
    "tasks": [
      {
        "name": "get_todo",
        "type": "http",
        "timeout": "2s",
        "input": {
          "method": "GET",
          "uri": "https://jsonplaceholder.typicode.com/todos/${context.input.todoId}"
        }
      },
      {
        "name": "get_user",
        "type": "http",
        "timeout": "2s",
        "input": {
          "method": "GET",
          "uri": "https://jsonplaceholder.typicode.com/users/${get_todo.output.body.userId}"
        }
      }
    ]
  }
}`)

	task, err := r.ConstructFromJSON(orchestrator.NewConstructDecoder(r), data)
	if err != nil {
		fmt.Println(err)
		return
	}

	input := orchestrator.NewInput(map[string]interface{}{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]interface{})
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}
