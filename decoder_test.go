package orchestrator_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/RussellLuo/orchestrator"
)

func TestDecoder_Decode(t *testing.T) {
	type Out struct {
		A string        `orchestrator:"a"`
		B int           `orchestrator:"b"`
		C string        `orchestrator:"c"`
		D string        `orchestrator:"d"`
		E string        `orchestrator:"e"`
		F time.Duration `orchestrator:"f"`
	}

	decoder := orchestrator.NewDecoder()
	decoder.AddInput("task1", map[string]interface{}{
		"value": "1",
	})
	decoder.AddOutput("task2", map[string]interface{}{
		"value": 2,
	})

	in := map[string]interface{}{
		"a": "${task1.input.value}",
		"b": "${task2.output.value}",
		"c": "/posts/${task2.output.value}",
		"d": "${task1.input.value}.${task2.output.value}",
		"e": "task.output.value",
		"f": "3s",
	}
	var out Out
	if err := decoder.Decode(in, &out); err != nil {
		t.Fatalf("Err: %v", err)
	}

	wantOut := Out{
		A: "1",
		B: 2,
		C: "/posts/2",
		D: "1.2",
		E: "task.output.value",
		F: 3 * time.Second,
	}
	if !reflect.DeepEqual(out, wantOut) {
		t.Fatalf("Out: Got (%#v) != Want (%#v)", out, wantOut)
	}
}
