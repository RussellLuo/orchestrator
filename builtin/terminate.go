package builtin

import (
	"context"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeTerminate = "terminate"
)

// Terminate is a leaf task that can terminate a series of tasks with a given output.
type Terminate struct {
	*orchestrator.TaskDefinition

	input struct {
		Output orchestrator.Output `orchestrator:"output"`
	}
}

func NewTerminate(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeTerminate {
		def.Type = TypeTerminate
	}

	return &Terminate{TaskDefinition: def}, nil
}

func (t *Terminate) Definition() *orchestrator.TaskDefinition {
	return t.TaskDefinition
}

func (t *Terminate) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	if err := decoder.Decode(t.InputTemplate, &t.input); err != nil {
		return nil, err
	}

	t.input.Output.SetTerminated()
	return t.input.Output, nil
}
