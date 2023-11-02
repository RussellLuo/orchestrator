package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeRaise = "raise"
)

func init() {
	MustRegisterRaise(orchestrator.GlobalRegistry)
}

func MustRegisterRaise(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeRaise,
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			raise := &Raise{def: def}
			if err := r.Decode(def.InputTemplate, &raise.Input); err != nil {
				return nil, err
			}
			return raise, nil
		},
	})
}

// Raise is a leaf task that is used to terminate the execution of a flow by raising an error.
type Raise struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Error orchestrator.Expr[string] `json:"error"`
	}
}

func NewRaise(name string) *Raise {
	return &Raise{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeRaise,
		},
	}
}

func (r *Raise) Error(err any) *Raise {
	r.Input.Error = orchestrator.Expr[string]{Expr: err}
	return r
}

func (r *Raise) Name() string { return r.def.Name }

func (r *Raise) String() string {
	return fmt.Sprintf("%s(name:%s, error:%v)", r.def.Type, r.def.Name, r.Input.Error.Expr)
}

func (r *Raise) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := r.Input.Error.Evaluate(input); err != nil {
		return nil, err
	}

	// Always return an error with the given message.
	errMessage := r.Input.Error.Value
	return nil, fmt.Errorf(errMessage)
}
