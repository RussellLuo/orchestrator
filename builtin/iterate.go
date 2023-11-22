package builtin

import (
	"context"
	"fmt"
	"sort"

	"github.com/RussellLuo/orchestrator"
)

type IterateType string

const (
	TypeIterate = "iterate"

	IterateTypeList  IterateType = "list"
	IterateTypeMap   IterateType = "map"
	IterateTypeRange IterateType = "range"
)

func init() {
	MustRegisterIterate(orchestrator.GlobalRegistry)
}

func MustRegisterIterate(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeIterate,
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Iterate{def: def}
			if err := r.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Iterate is a leaf task that is used to make an iterator from a slice/map/range.
// Note that an Iterate task is always used along with a Loop task.
type Iterate struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Type  IterateType `json:"type"`
		Value any         `json:"value"`
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

func (i *Iterate) List(v any) *Iterate {
	i.Input.Type = IterateTypeList
	i.Input.Value = v
	return i
}

func (i *Iterate) Map(v any) *Iterate {
	i.Input.Type = IterateTypeMap
	i.Input.Value = v
	return i
}

func (i *Iterate) Range(v any) *Iterate {
	i.Input.Type = IterateTypeRange
	i.Input.Value = v
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
	if i.Input.Value == nil {
		return nil, fmt.Errorf("bad iterate value")
	}

	var value any
	switch i.Input.Type {
	case IterateTypeList:
		expr := orchestrator.Expr[[]any]{Expr: i.Input.Value}
		if err := expr.Evaluate(input); err != nil {
			return nil, err
		}
		value = expr.Value

	case IterateTypeMap:
		expr := orchestrator.Expr[map[string]any]{Expr: i.Input.Value}
		if err := expr.Evaluate(input); err != nil {
			return nil, err
		}
		value = expr.Value

	case IterateTypeRange:
		expr := orchestrator.Expr[[]int]{Expr: i.Input.Value}
		if err := expr.Evaluate(input); err != nil {
			return nil, err
		}
		switch len(expr.Value) {
		case 2, 3:
		default:
			return nil, fmt.Errorf("bad iterate value length: want 2 or 3 but got %d", len(expr.Value))
		}
		value = expr.Value

	default:
		return nil, fmt.Errorf(`bad iterate type: must be one of [%q, %q, %q]`, IterateTypeList, IterateTypeMap, IterateTypeRange)
	}

	iterator := orchestrator.NewIterator(ctx, func(sender *orchestrator.IteratorSender) {
		defer sender.End() // End the iteration

		switch i.Input.Type {
		case IterateTypeList:
			vList := value.([]any)
			for _, v := range vList {
				if continue_ := sender.Send(orchestrator.Output{"value": v}, nil); !continue_ {
					return
				}
			}

		case IterateTypeMap:
			vMap := value.(map[string]any)

			// Sort map keys in ascending order.
			var keys []string
			for k := range vMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				if continue_ := sender.Send(orchestrator.Output{"key": k, "value": vMap[k]}, nil); !continue_ {
					return
				}
			}

		case IterateTypeRange:
			var start, stop, step int
			vRange := value.([]int)
			switch len(vRange) {
			case 2:
				start, stop, step = vRange[0], vRange[1], 1
			case 3:
				start, stop, step = vRange[0], vRange[1], vRange[2]
			}
			for n := start; n < stop; n += step {
				if continue_ := sender.Send(orchestrator.Output{"value": n}, nil); !continue_ {
					return
				}
			}
		}
	})
	return orchestrator.Output{"iterator": iterator}, nil
}
