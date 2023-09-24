package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeIterate = "iterate"
)

func init() {
	MustRegisterIterate(orchestrator.GlobalRegistry)
}

func MustRegisterIterate(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeIterate,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Iterate{def: def}
			if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Iterate is a leaf task that is used to make an iterator from a slice/map/range.
// Note that an Iterate task is always used with a Loop task.
type Iterate struct {
	def *orchestrator.TaskDefinition

	Input struct {
		In    orchestrator.Expr[any]   `json:"in,omitempty"`
		Range orchestrator.Expr[[]int] `json:"range,omitempty"`
	}
}

func NewIterate(name string) *Iterate {
	return &Iterate{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeIterate,
		},
	}
}

func (i *Iterate) In(in any) *Iterate {
	i.Input.In = orchestrator.Expr[any]{Expr: in}
	return i
}

func (i *Iterate) Range(range_ any) *Iterate {
	i.Input.Range = orchestrator.Expr[[]int]{Expr: range_}
	return i
}

func (i *Iterate) Name() string { return i.def.Name }

func (i *Iterate) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s)",
		i.def.Type,
		i.def.Name,
		i.def.Timeout,
	)
}

func (i *Iterate) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if i.Input.In.Expr == nil && i.Input.Range.Expr == nil {
		return nil, fmt.Errorf("neither 'in' nor 'range' was set")
	}

	if i.Input.In.Expr != nil && i.Input.Range.Expr != nil {
		return nil, fmt.Errorf("only one of 'in' and 'range' can be set")
	}

	if i.Input.In.Expr != nil {
		if err := i.Input.In.Evaluate(input); err != nil {
			return nil, err
		}
	}

	if i.Input.Range.Expr != nil {
		if err := i.Input.Range.Evaluate(input); err != nil {
			return nil, err
		}
	}

	iterator := NewIterator(func(ctx context.Context, ch chan<- Result) {
		send := func(output orchestrator.Output, err error) (continue_ bool) {
			select {
			case ch <- Result{Output: output, Err: err}:
				return true
			case <-ctx.Done():
				return false
			}
		}

		if i.Input.In.Value != nil {
			switch value := i.Input.In.Value.(type) {
			case []any:
				for _, v := range value {
					if continue_ := send(orchestrator.Output{"value": v}, nil); !continue_ {
						return
					}
				}
			case map[string]any:
				for k, v := range value {
					if continue_ := send(orchestrator.Output{"key": k, "value": v}, nil); !continue_ {
						return
					}
				}
			default:
				send(nil, fmt.Errorf("bad in: want slice or map but got %T", value))
				return
			}
		}

		if len(i.Input.Range.Value) > 0 {
			value := i.Input.Range.Value
			var start, stop, step int

			switch len(value) {
			case 2:
				start, stop, step = value[0], value[1], 1
			case 3:
				start, stop, step = value[0], value[1], value[2]
			default:
				send(nil, fmt.Errorf("bad range length: want 2 or 3 but got %d", len(value)))
				return
			}
			for n := start; n < stop; n += step {
				if continue_ := send(orchestrator.Output{"value": n}, nil); !continue_ {
					return
				}
			}
		}

		// End the iteration.
		close(ch)
	})
	return orchestrator.Output{"iterator": iterator}, nil
}
