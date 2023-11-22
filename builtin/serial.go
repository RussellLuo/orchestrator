package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeSerial = "serial"
)

func init() {
	MustRegisterSerial(orchestrator.GlobalRegistry)
}

func MustRegisterSerial(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeSerial,
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Serial{def: def}
			if err := r.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

func executeWithTimeout(ctx context.Context, input orchestrator.Input, timeout time.Duration, f func(context.Context, orchestrator.Input) (orchestrator.Output, error)) (orchestrator.Output, error) {
	if timeout <= 0 {
		// Execute f directly.
		return f(ctx, input)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan orchestrator.Result, 1)
	go func() {
		output, err := f(ctx, input)
		resultChan <- orchestrator.Result{Output: output, Err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result.Output, result.Err
	}
}

// Serial is a composite task that is used to execute its subtasks serially.
type Serial struct {
	def *orchestrator.TaskDefinition

	Input struct {
		// The optional schema for the following series of subtasks.
		//
		// Typically, the schema is required for a standalone workflow.
		Schema orchestrator.Schema `json:schema,omitempty`

		Tasks []orchestrator.Task `json:"tasks"`
	}
}

func NewSerial(name string) *Serial {
	return &Serial{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeSerial,
		},
	}
}

func (s *Serial) Timeout(timeout time.Duration) *Serial {
	s.def.Timeout = timeout
	return s
}

func (s *Serial) Tasks(tasks ...orchestrator.Task) *Serial {
	s.Input.Tasks = tasks
	return s
}

func (s *Serial) Name() string { return s.def.Name }

func (s *Serial) String() string {
	var inputStrings []string
	for _, t := range s.Input.Tasks {
		inputStrings = append(inputStrings, t.String())
	}
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, tasks:[%s])",
		s.def.Type,
		s.def.Name,
		s.def.Timeout,
		strings.Join(inputStrings, ", "),
	)
}

func (s *Serial) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	// Validate the external input against the schema.
	if err := s.Input.Schema.Validate(input.Get("input")); err != nil {
		return nil, err
	}
	return executeWithTimeout(ctx, input, s.def.Timeout, s.execute)
}

func (s *Serial) execute(ctx context.Context, input orchestrator.Input) (output orchestrator.Output, err error) {
	trace := orchestrator.TraceFromContext(ctx).New(s.Name())
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	for _, t := range s.Input.Tasks {
		output, err = trace.Wrap(t).Execute(ctx, input)
		if err != nil {
			return nil, err
		}

		if output.IsTerminated() {
			return output, nil
		}

		input.Add(t.Name(), output)
	}

	return output, nil
}
