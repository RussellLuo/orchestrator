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
			"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
		),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}",
		),
	)

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}

func Example_trace() {
	task := builtin.NewSerial("get_todo_user").Timeout(3*time.Second).Tasks(
		builtin.NewHTTP("get_todo").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
		),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}",
		),
	)

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	event := orchestrator.TraceTask(context.Background(), task, input)

	// Note that for the stability of the test, we just show the output.
	// You may be interested in other properties of the tracing event.
	body := event.Output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}

func Example_constructFromJSON() {
	r := orchestrator.NewRegistry()
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
          "uri": "https://jsonplaceholder.typicode.com/todos/${input.todoId}"
        }
      },
      {
        "name": "get_user",
        "type": "http",
        "timeout": "2s",
        "input": {
          "method": "GET",
          "uri": "https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}"
        }
      }
    ]
  }
}`)

	task, err := r.ConstructFromJSON(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}
