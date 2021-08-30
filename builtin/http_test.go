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
	task, err := builtin.NewHTTP(&orchestrator.TaskDefinition{
		Name:    "test",
		Timeout: 2 * time.Second,
		InputTemplate: orchestrator.InputTemplate{
			"method": "GET",
			"uri":    "https://jsonplaceholder.typicode.com/todos/1",
		},
	})
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	decoder := orchestrator.NewDecoder()
	output, err := task.Execute(context.Background(), decoder)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	wantOutput := orchestrator.Output{
		"status": 200,
		"body": map[string]interface{}{
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
