package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeDecision = "decision"
)

func init() {
	MustRegisterDecision(orchestrator.GlobalRegistry)
}

func MustRegisterDecision(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeDecision,
		New:  func() orchestrator.Task { return new(Decision) },
	})
}

// Decision is a composite task that is similar to the `switch` statement in Go.
type Decision struct {
	orchestrator.TaskHeader

	Input struct {
		Expression orchestrator.Expr[any]    `json:"expression"`
		Cases      map[any]orchestrator.Task `json:"cases"`
		Default    orchestrator.Task         `json:"default"`
	} `json:"input"`
}

func NewDecision(name string) *Decision {
	return &Decision{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeDecision,
		},
	}
}

func (d *Decision) Timeout(timeout time.Duration) *Decision {
	d.TaskHeader.Timeout = timeout
	return d
}

func (d *Decision) Expression(s any) *Decision {
	d.Input.Expression = orchestrator.Expr[any]{Expr: s}
	return d
}

func (d *Decision) Case(c any, task orchestrator.Task) *Decision {
	if d.Input.Cases == nil {
		d.Input.Cases = make(map[any]orchestrator.Task)
	}
	d.Input.Cases[c] = task
	return d
}

func (d *Decision) Default(task orchestrator.Task) *Decision {
	d.Input.Default = task
	return d
}

func (d *Decision) String() string {
	casesInputStrings := make(map[any]string)
	for v, t := range d.Input.Cases {
		casesInputStrings[v] = t.String()
	}

	var defaultInputString string
	if d.Input.Default != nil {
		defaultInputString = d.Input.Default.String()
	}

	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, expression:%v, cases:%v, default:%s)",
		d.TaskHeader.Type,
		d.TaskHeader.Name,
		d.TaskHeader.Timeout,
		d.Input.Expression.Expr,
		casesInputStrings,
		defaultInputString,
	)
}

func (d *Decision) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(d.Name)
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	if err := d.Input.Expression.Evaluate(input); err != nil {
		return nil, err
	}

	task, ok := d.Input.Cases[d.Input.Expression.Value]
	if !ok {
		if d.Input.Default != nil {
			return trace.Wrap(d.Input.Default).Execute(ctx, input)
		}
		return nil, nil
	}

	return trace.Wrap(task).Execute(ctx, input)
}
