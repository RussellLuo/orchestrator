package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeTerminate = "terminate"
)

func init() {
	MustRegisterTerminate(orchestrator.GlobalRegistry)
}

func MustRegisterTerminate(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeTerminate,
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Terminate{def: def}
			if err := r.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Terminate is a leaf task that can terminate a series of tasks with a given output.
type Terminate struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Output orchestrator.Expr[orchestrator.Output] `json:"output"`
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

func (t *Terminate) Output(output any) *Terminate {
	t.Input.Output = orchestrator.Expr[orchestrator.Output]{Expr: output}
	return t
}

func (t *Terminate) Name() string { return t.def.Name }

func (t *Terminate) String() string {
	return fmt.Sprintf("%s(name:%s, output:%v)", t.def.Type, t.def.Name, t.Input.Output.Expr)
}

func (t *Terminate) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := t.Input.Output.Evaluate(input); err != nil {
		return nil, err
	}

	output := t.Input.Output.Value
	output.SetTerminated()
	return output, nil
}
