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
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			w := &Wait{def: def}
			if err := r.Decode(def.InputTemplate, &w.Input); err != nil {
				return nil, err
			}
			return w, nil
		},
	})
}

// Wait is a leaf task that is used to wait for receiving an external input
// (and sometimes also send an output externally before that).
//
// Note that a Wait task must be used within an actor (i.e. asynchronous Serial task).
type Wait struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Output      orchestrator.Expr[map[string]any] `json:"output"`
		InputSchema map[string]any                    `json:"input_schema"`
	}
}

func NewWait(name string) *Wait {
	return &Wait{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeWait,
		},
	}
}

func (w *Wait) Name() string { return w.def.Name }

func (w *Wait) String() string {
	return fmt.Sprintf("%s(name:%s)", w.def.Type, w.def.Name)
}

func (w *Wait) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := w.Input.Output.Evaluate(input); err != nil {
		return nil, err
	}

	behavior, ok := input.Get("actor")["behavior"].(*orchestrator.ActorBehavior)
	if !ok {
		return nil, fmt.Errorf("task %q (of type Wait) must be used within an asynchronous flow", w.Name())
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
