package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeFunc = "func"
)

func init() {
	MustRegisterFunc(orchestrator.GlobalRegistry)
}

func MustRegisterFunc(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeFunc,
		New:  func() orchestrator.Task { return new(Func) },
	})
}

// Func is a leaf task that is used to execute the input function with the given arguments.
type Func struct {
	orchestrator.TaskHeader

	Input struct {
		Func func(context.Context, orchestrator.Input) (orchestrator.Output, error) `json:"func"`
	} `json:input`
}

func NewFunc(name string) *Func {
	return &Func{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeFunc,
		},
	}
}

func (f *Func) Func(ef func(context.Context, orchestrator.Input) (orchestrator.Output, error)) *Func {
	f.Input.Func = ef
	return f
}

func (f *Func) String() string {
	return fmt.Sprintf("%s(name:%s)", f.TaskHeader.Type, f.TaskHeader.Name)
}

func (f *Func) Execute(ctx context.Context, input orchestrator.Input) (output orchestrator.Output, err error) {
	return f.Input.Func(ctx, input)
}
