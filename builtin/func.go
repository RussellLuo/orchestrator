package builtin

import (
	"context"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeFunc = "func"
)

// Func is a leaf task that is used to execute the input function with the given arguments.
type Func struct {
	*orchestrator.TaskDefinition

	input struct {
		Func func(context.Context, *orchestrator.Decoder) (orchestrator.Output, error) `orchestrator:"func"`
	}
}

func NewFunc(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeFunc {
		def.Type = TypeFunc
	}

	f := &Func{TaskDefinition: def}
	decoder := orchestrator.NewDecoder().NoRendering()
	if err := decoder.Decode(def.InputTemplate, &f.input); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Func) Definition() *orchestrator.TaskDefinition {
	return f.TaskDefinition
}

func (f *Func) Execute(ctx context.Context, decoder *orchestrator.Decoder) (output orchestrator.Output, err error) {
	return f.input.Func(ctx, decoder)
}
