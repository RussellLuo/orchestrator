package builtin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/orchestrator/builtin"
)

func TestHTTP_Execute(t *testing.T) {
	task := builtin.NewHTTP("test").Timeout(2 * time.Second).Get(
		"https://jsonplaceholder.typicode.com/todos/${input.todoId}",
	)

	input := orchestrator.NewInput(map[string]any{"todoId": 1})
	output, err := task.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	wantOutput := orchestrator.Output{
		"status": 200,
		"body": map[string]any{
			"userId":    1,
			"id":        1,
			"title":     "delectus aut autem",
			"completed": false,
		},
	}
	if output["status"] != wantOutput["status"] {
		t.Fatalf("Status: Got (%#v) != Want (%#v)", output["status"], wantOutput["status"])
	}
	if fmt.Sprintf("%#v", output["body"]) != fmt.Sprintf("%#v", wantOutput["body"]) {
		t.Fatalf("Body: Got (%#v) != Want (%#v)", output["body"], wantOutput["body"])
	}
}
