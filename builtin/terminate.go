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
		New:  func() orchestrator.Task { return new(Terminate) },
	})
}

// Terminate is a leaf task that is used to terminate the execution of a flow and return an output.
type Terminate struct {
	orchestrator.TaskHeader

	Input struct {
		Output orchestrator.Expr[orchestrator.Output] `json:"output"`
		Error  orchestrator.Expr[string]              `json:"error"`
	} `json:"input"`
}

func (t *Terminate) String() string {
	return fmt.Sprintf("%s(name:%s, output:%v, error:%v)", t.Type, t.Name, t.Input.Output.Expr, t.Input.Error.Expr)
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

type TerminateBuilder struct {
	task *Terminate
}

func NewTerminate(name string) *TerminateBuilder {
	task := &Terminate{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeTerminate,
		},
	}
	return &TerminateBuilder{task: task}
}

func (b *TerminateBuilder) Output(output any) *TerminateBuilder {
	b.task.Input.Output = orchestrator.Expr[orchestrator.Output]{Expr: output}
	return b
}

func (b *TerminateBuilder) Error(err any) *TerminateBuilder {
	b.task.Input.Error = orchestrator.Expr[string]{Expr: err}
	return b
}

func (b *TerminateBuilder) Build() orchestrator.Task {
	return b.task
}
