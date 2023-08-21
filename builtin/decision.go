package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeDecision = "decision"
)

func init() {
	MustRegisterDecision(orchestrator.GlobalRegistry)
}

func MustRegisterDecision(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeDecision,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Decision{def: def}
			if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Decision is a composite task that is similar to the `switch` statement in Go.
type Decision struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Switch  interface{}                       `orchestrator:"switch"`
		Cases   map[interface{}]orchestrator.Task `orchestrator:"cases"`
		Default orchestrator.Task                 `orchestrator:"default"`
	}

	Expression struct {
		Switch interface{} `orchestrator:"switch"`
	}
}

func NewDecision(name string) *Decision {
	return &Decision{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeDecision,
		},
	}
}

func (d *Decision) Timeout(timeout time.Duration) *Decision {
	d.def.Timeout = timeout
	return d
}

func (d *Decision) Switch(s interface{}) *Decision {
	d.Input.Switch = s
	return d
}

func (d *Decision) Case(c interface{}, task orchestrator.Task) *Decision {
	if d.Input.Cases == nil {
		d.Input.Cases = make(map[interface{}]orchestrator.Task)
	}
	d.Input.Cases[c] = task
	return d
}

func (d *Decision) Default(task orchestrator.Task) *Decision {
	d.Input.Default = task
	return d
}

func (d *Decision) Name() string { return d.def.Name }

func (d *Decision) String() string {
	casesInputStrings := make(map[interface{}]string)
	for v, t := range d.Input.Cases {
		casesInputStrings[v] = t.String()
	}

	var defaultInputString string
	if d.Input.Default != nil {
		defaultInputString = d.Input.Default.String()
	}

	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, switch:%v, cases:%v, default:%s)",
		d.def.Type,
		d.def.Name,
		d.def.Timeout,
		d.Input.Switch,
		casesInputStrings,
		defaultInputString,
	)
}

func (d *Decision) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := input.Decoder.Decode(d.Input, &d.Expression); err != nil {
		return nil, err
	}

	task, ok := d.Input.Cases[d.Expression.Switch]
	if !ok {
		if d.Input.Default != nil {
			return d.Input.Default.Execute(ctx, input)
		}
		return nil, nil
	}

	return task.Execute(ctx, input)
}
