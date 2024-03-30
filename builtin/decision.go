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

type DecisionBuilder struct {
	task *Decision
}

func NewDecision(name string) *DecisionBuilder {
	task := &Decision{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeDecision,
		},
	}
	return &DecisionBuilder{task: task}
}

func (b *DecisionBuilder) Timeout(timeout time.Duration) *DecisionBuilder {
	b.task.TaskHeader.Timeout = timeout
	return b
}

func (b *DecisionBuilder) Expression(s any) *DecisionBuilder {
	b.task.Input.Expression = orchestrator.Expr[any]{Expr: s}
	return b
}

func (b *DecisionBuilder) Case(c any, builder orchestrator.Builder) *DecisionBuilder {
	if b.task.Input.Cases == nil {
		b.task.Input.Cases = make(map[any]orchestrator.Task)
	}
	b.task.Input.Cases[c] = builder.Build()
	return b
}

func (b *DecisionBuilder) Default(builder orchestrator.Builder) *DecisionBuilder {
	b.task.Input.Default = builder.Build()
	return b
}

func (b *DecisionBuilder) Build() orchestrator.Task {
	return b.task
}
