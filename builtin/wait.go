package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeWait = "wait"
)

func init() {
	MustRegisterWait(orchestrator.GlobalRegistry)
}

func MustRegisterWait(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeWait,
		New:  func() orchestrator.Task { return new(Wait) },
	})
}

// Wait is a leaf task that is used to wait for receiving an external input
// (and sometimes also send an output externally before that).
//
// Note that a Wait task must be used within an actor (i.e. asynchronous Serial task).
type Wait struct {
	orchestrator.TaskHeader

	Input struct {
		Output      orchestrator.Expr[map[string]any] `json:"output"`
		InputSchema map[string]any                    `json:"input_schema"`
	} `json:"input"`
}

func (w *Wait) String() string {
	return fmt.Sprintf("%s(name:%s)", w.Type, w.Name)
}

func (w *Wait) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := w.Input.Output.Evaluate(input); err != nil {
		return nil, err
	}

	behavior, ok := input.Get("actor")["behavior"].(*orchestrator.ActorBehavior)
	if !ok {
		return nil, fmt.Errorf("task %q (of type Wait) must be used within an asynchronous flow", w.Name)
	}

	// Send the output value, if non-empty, to the actor's outbox.
	if len(w.Input.Output.Value) > 0 {
		data := map[string]any{
			"output":       w.Input.Output.Value,
			"input_schema": w.Input.InputSchema,
			"status":       "pause", // Mark the actor status as "pause".
		}
		behavior.Send(data, nil)
	}

	// Receive the input value from the actor's inbox.
	receivedInput := behavior.Receive()
	if receivedInput == nil {
		return nil, fmt.Errorf("execution has been canceled")
	}

	// Validate the received input against the schema.
	schema := orchestrator.Schema{Input: w.Input.InputSchema}
	if err := schema.Validate(receivedInput); err != nil {
		return nil, err
	}

	return orchestrator.Output{"input": receivedInput}, nil
}

type WaitBuilder struct {
	task *Wait
}

func NewWait(name string) *WaitBuilder {
	task := &Wait{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeWait,
		},
	}
	return &WaitBuilder{task: task}
}

func (b *WaitBuilder) Output(output map[string]any) *WaitBuilder {
	b.task.Input.Output = orchestrator.Expr[map[string]any]{Expr: output}
	return b
}

func (b *WaitBuilder) Build() orchestrator.Task {
	return b.task
}
