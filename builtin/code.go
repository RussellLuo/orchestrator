package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeCode = "code"
)

func init() {
	MustRegisterCode(orchestrator.GlobalRegistry)
}

func MustRegisterCode(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeCode,
		New:  func() orchestrator.Task { return new(Code) },
	})
}

// Code is a leaf task that is used to execute a snippet of Starlark code.
//
// Note that the signature of the function must be `def _(env):`, where `env` is the
// environment that contains the input and outputs of all the previously executed tasks.
//
// Examples:
//
//	def _(env):
//	    return [x*2 for x in env.input.values]
type Code struct {
	orchestrator.TaskHeader

	Input struct {
		Code string `json:"code"`
	} `json:"input"`
}

func (c *Code) String() string {
	return fmt.Sprintf("%s(name:%s)", c.TaskHeader.Type, c.TaskHeader.Name)
}

func (c *Code) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	result, err := orchestrator.StarlarkCallFunc(c.Input.Code, input.Env())
	if err != nil {
		return nil, err
	}

	// If the result is a map, return it as the output.
	if r, ok := result.(map[string]any); ok {
		return orchestrator.Output(r), nil
	}
	// Otherwise, create an output that contains only one field "result".
	return orchestrator.Output{"result": result}, nil
}

type CodeBuilder struct {
	task *Code
}

func NewCode(name string) *CodeBuilder {
	task := &Code{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeCode,
		},
	}
	return &CodeBuilder{task: task}
}

func (b *CodeBuilder) Code(s string) *CodeBuilder {
	b.task.Input.Code = s
	return b
}

func (b *CodeBuilder) Build() orchestrator.Task {
	return b.task
}
