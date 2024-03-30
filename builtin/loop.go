package builtin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeLoop = "loop"
)

func init() {
	MustRegisterLoop(orchestrator.GlobalRegistry)
}

func MustRegisterLoop(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeLoop,
		New:  func() orchestrator.Task { return new(Loop) },
	})
}

// Loop is a composite task that is similar to the `for` statement in Go.
type Loop struct {
	orchestrator.TaskHeader

	Input struct {
		Iterator orchestrator.Task `json:"iterator"`
		Body     orchestrator.Task `json:"body"`
	} `json:"input"`
}

func (l *Loop) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s)",
		l.Type,
		l.Name,
		l.Timeout,
	)
}

func (l *Loop) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(l.Name)
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	iterOutput, err := trace.Wrap(l.Input.Iterator).Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	iterName := l.Input.Iterator.Header().Name
	iter, ok := iterOutput.Iterator()
	if !ok {
		return nil, fmt.Errorf("bad iterator: %s", iterName)
	}

	output := make(orchestrator.Output)

	var i int
	for result := range iter.Next() {
		if result.Err != nil {
			return nil, result.Err
		}
		// Set the output of the iterator task for the current iteration.
		input.Add(iterName, result.Output)
		o, err := trace.Wrap(l.Input.Body).Execute(ctx, input)
		if err != nil {
			return nil, err
		}

		// Save the output of the body task for the current iteration.
		output[strconv.Itoa(i)] = map[string]any(o)
		i++

		if o.IsTerminated() {
			// Break the iteration.
			iter.Break()
			goto End
		}
	}

End:
	// Save the total iteration number.
	output["iteration"] = i
	return output, nil
}

type LoopBuilder struct {
	task *Loop
}

func NewLoop(name string) *LoopBuilder {
	task := &Loop{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeLoop,
		},
	}
	return &LoopBuilder{task: task}
}

func (b *LoopBuilder) Timeout(timeout time.Duration) *LoopBuilder {
	b.task.Timeout = timeout
	return b
}

func (b *LoopBuilder) Iterator(builder orchestrator.Builder) *LoopBuilder {
	b.task.Input.Iterator = builder.Build()
	return b
}

func (b *LoopBuilder) Body(builder orchestrator.Builder) *LoopBuilder {
	b.task.Input.Body = builder.Build()
	return b
}

func (b *LoopBuilder) Build() orchestrator.Task {
	return b.task
}
