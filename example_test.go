package orchestrator_test

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func Example() {
	flow := builtin.NewSerial("get_todo_user").Timeout(3*time.Second).Tasks(
		builtin.NewHTTP("get_todo").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
		),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}",
		),
	).Build()

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := flow.Execute(context.Background(), input)
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
	flow := builtin.NewSerial("get_todo_user").Timeout(3*time.Second).Tasks(
		builtin.NewHTTP("get_todo").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
		),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}",
		),
	).Build()

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	event := orchestrator.TraceTask(context.Background(), flow, input)

	// Note that for the stability of the test, we just show the output.
	// You may be interested in other properties of the tracing event.
	body := event.Output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}

func Example_JSON() {
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

	flow, err := r.ConstructFromJSON(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := flow.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Leanne Graham
}

func Example_actor() {
	flow := builtin.NewSerial("get_todo_user").Async(true).Tasks(
		builtin.NewHTTP("get_todo").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
		),
		builtin.NewFunc("echo_once").Func(func(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
			behavior, ok := input.Get("actor")["behavior"].(*orchestrator.ActorBehavior)
			if !ok {
				return nil, fmt.Errorf("task %q (of type Interact) must be used within an asynchronous flow", "echo_once")
			}

			// Send the data, received from the actor's inbox, to the actor's outbox.
			data := behavior.Receive()
			behavior.Send(data, nil)

			return orchestrator.Output{}, nil
		}),
		builtin.NewHTTP("get_user").Timeout(2*time.Second).Get(
			"https://jsonplaceholder.typicode.com/users/${get_todo.body.userId}",
		),
	).Build()

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := flow.Execute(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return
	}

	actor, ok := output.Actor()
	if !ok {
		fmt.Println("bad actor")
		return
	}

	// Perform a ping-pong action midway.
	actor.Inbox() <- map[string]any{"data": "Hello"}
	result := <-actor.Outbox()
	fmt.Println(result.Output["data"]) // Ignore error handling for simplicity.

	// Finally, get the flow result.
	result = <-actor.Outbox()
	if result.Err != nil {
		fmt.Println(result.Err)
		return
	}

	body := result.Output["body"].(map[string]any)
	fmt.Println(body["name"])

	// Output:
	// Hello
	// Leanne Graham
}
