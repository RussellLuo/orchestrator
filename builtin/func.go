package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeFunc = "func"
)

func init() {
	MustRegisterFunc(orchestrator.GlobalRegistry)
}

func MustRegisterFunc(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeFunc,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Func{def: def}
			if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Func is a leaf task that is used to execute the input function with the given arguments.
type Func struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Func func(context.Context, *orchestrator.Decoder) (orchestrator.Output, error) `orchestrator:"func"`
	}
}

func NewFunc(name string) *Func {
	return &Func{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeFunc,
		},
	}
}

func (f *Func) Func(ef func(context.Context, *orchestrator.Decoder) (orchestrator.Output, error)) *Func {
	f.Input.Func = ef
	return f
}

func (f *Func) InputString() string {
	return fmt.Sprintf("%s(name:%s)", f.def.Type, f.def.Name)
}

func (f *Func) Definition() *orchestrator.TaskDefinition {
	return f.def
}

func (f *Func) Execute(ctx context.Context, decoder *orchestrator.Decoder) (output orchestrator.Output, err error) {
	return f.Input.Func(ctx, decoder)
}
