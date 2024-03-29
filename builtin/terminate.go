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

// Terminate is a leaf task that is used to terminate the execution of a flow and return an output.
type Terminate struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Output orchestrator.Expr[orchestrator.Output] `json:"output"`
		Error  orchestrator.Expr[string]              `json:"error"`
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

func (t *Terminate) Error(err any) *Terminate {
	t.Input.Error = orchestrator.Expr[string]{Expr: err}
	return t
}

func (t *Terminate) Name() string { return t.def.Name }

func (t *Terminate) String() string {
	return fmt.Sprintf("%s(name:%s, output:%v, error:%v)", t.def.Type, t.def.Name, t.Input.Output.Expr, t.Input.Error.Expr)
}

func (t *Terminate) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := t.Input.Output.Evaluate(input); err != nil {
		return nil, err
	}
	if err := t.Input.Error.Evaluate(input); err != nil {
		return nil, err
	}

	// If specified, return an error with the given message.
	errMessage := t.Input.Error.Value
	if errMessage != "" {
		return nil, fmt.Errorf(errMessage)
	}

	// Otherwise, return a normal output.
	output := orchestrator.Output{}
	for k, v := range t.Input.Output.Value {
		output[k] = v
	}
	output.SetTerminated()
	return output, nil
}
