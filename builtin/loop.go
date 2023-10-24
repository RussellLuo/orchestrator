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
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Loop{def: def}
			if err := r.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Loop is a composite task that is similar to the `for` statement in Go.
type Loop struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Iterator orchestrator.Task `json:"iterator"`
		Body     orchestrator.Task `json:"body"`
	}
}

func NewLoop(name string) *Loop {
	return &Loop{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeLoop,
		},
	}
}

func (l *Loop) Timeout(timeout time.Duration) *Loop {
	l.def.Timeout = timeout
	return l
}

func (l *Loop) Iterator(task orchestrator.Task) *Loop {
	l.Input.Iterator = task
	return l
}

func (l *Loop) Body(task orchestrator.Task) *Loop {
	l.Input.Body = task
	return l
}

func (l *Loop) Name() string { return l.def.Name }

func (l *Loop) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s)",
		l.def.Type,
		l.def.Name,
		l.def.Timeout,
	)
}

func (l *Loop) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(l.Name())
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	iterOutput, err := trace.Wrap(l.Input.Iterator).Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	iterName := l.Input.Iterator.Name()
	iter, ok := iterOutput["iterator"].(*Iterator)
	if !ok {
		return nil, fmt.Errorf("bad iterator: %s", iterName)
	}

	iter.Start()
	defer iter.Stop()

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
	}

	// Save the total iteration number.
	output["iteration"] = i
	return output, nil
}

type Iterator struct {
	f      func(context.Context, chan<- Result)
	cancel func()

	c chan Result
}

func NewIterator(f func(context.Context, chan<- Result)) *Iterator {
	return &Iterator{
		f: f,
		c: make(chan Result),
	}
}

func (i *Iterator) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel
	go i.f(ctx, i.c)
}

func (i *Iterator) Stop() {
	i.cancel()
}

func (i *Iterator) Next() <-chan Result {
	return i.c
}

func (i *Iterator) String() string {
	return "<Iterator>"
}
