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
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			code := &Code{def: def}
			if err := r.Decode(def.InputTemplate, &code.Input); err != nil {
				return nil, err
			}
			return code, nil
		},
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
//	    return [x*2 for x in input.values]
type Code struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Code string `json:"code"`
	}
}

func NewCode(name string) *Code {
	return &Code{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeCode,
		},
	}
}

func (c *Code) Code(s string) *Code {
	c.Input.Code = s
	return c
}

func (c *Code) Name() string { return c.def.Name }

func (c *Code) String() string {
	return fmt.Sprintf("%s(name:%s)", c.def.Type, c.def.Name)
}

func (c *Code) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	result, err := orchestrator.StarlarkCallFunc(c.Input.Code, input.Env())
	if err != nil {
		return nil, err
	}
	return orchestrator.Output{"result": result}, nil
}
