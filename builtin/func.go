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

func (f *Func) String() string {
	return fmt.Sprintf("%s(name:%s)", f.Type, f.Name)
}

func (f *Func) Execute(ctx context.Context, input orchestrator.Input) (output orchestrator.Output, err error) {
	return f.Input.Func(ctx, input)
}

type FuncBuilder struct {
	task *Func
}

func NewFunc(name string) *FuncBuilder {
	task := &Func{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeFunc,
		},
	}
	return &FuncBuilder{task: task}
}

func (b *FuncBuilder) Func(ef func(context.Context, orchestrator.Input) (orchestrator.Output, error)) *FuncBuilder {
	b.task.Input.Func = ef
	return b
}

func (b *FuncBuilder) Build() orchestrator.Task {
	return b.task
}
