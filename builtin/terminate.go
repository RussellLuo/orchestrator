package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeTerminate = "terminate"
)

// Terminate is a leaf task that can terminate a series of tasks with a given output.
type Terminate struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Output orchestrator.Output `orchestrator:"output"`
	}
}

func NewTerminate(name string) *Terminate {
	return &Terminate{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeTerminate,
		},
	}
}

func (t *Terminate) Output(output orchestrator.Output) *Terminate {
	t.Input.Output = output
	return t
}

func (t *Terminate) InputString() string {
	return fmt.Sprintf("%s(name:%s, output:%v)", t.def.Type, t.def.Name, t.Input.Output)
}

func (t *Terminate) Definition() *orchestrator.TaskDefinition {
	return t.def
}

func (t *Terminate) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	output := orchestrator.Output{}
	if err := decoder.Decode(t.Input.Output, &output); err != nil {
		return nil, err
	}

	output.SetTerminated()
	return output, nil
}
