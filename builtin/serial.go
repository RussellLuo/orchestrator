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
		New:  func() orchestrator.Task { return new(Serial) },
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
	orchestrator.TaskHeader

	Input struct {
		Async bool `json:"async"`

		// The optional schema for the following series of subtasks.
		//
		// Typically, the schema is required for a standalone workflow.
		Schema orchestrator.Schema `json:schema,omitempty`

		Tasks []orchestrator.Task `json:"tasks"`
	} `json:"input"`
}

func (s *Serial) String() string {
	var inputStrings []string
	for _, t := range s.Input.Tasks {
		inputStrings = append(inputStrings, t.String())
	}
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, tasks:[%s])",
		s.Type,
		s.Name,
		s.Timeout,
		strings.Join(inputStrings, ", "),
	)
}

func (s *Serial) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	// Validate the external input against the schema.
	if err := s.Input.Schema.Validate(input.Get("input")); err != nil {
		return nil, err
	}

	if s.Input.Async {
		actor := orchestrator.NewActor(func(ctx context.Context, ab *orchestrator.ActorBehavior) {
			// Add the actor behavior into the input environment for later use.
			input.Add("actor", map[string]any{"behavior": ab})

			output, err := s.execute(ctx, input)
			if err != nil {
				ab.Send(nil, err)
				return
			}

			output["status"] = "finish" // Mark the actor status as "finish".
			ab.Send(output, nil)
		})
		return orchestrator.Output{"actor": actor}, nil
	}

	return executeWithTimeout(ctx, input, s.Timeout, s.execute)
}

func (s *Serial) execute(ctx context.Context, input orchestrator.Input) (output orchestrator.Output, err error) {
	trace := orchestrator.TraceFromContext(ctx).New(s.Name)
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	for _, t := range s.Input.Tasks {
		output, err = trace.Wrap(t).Execute(ctx, input)
		if err != nil {
			return nil, err
		}

		if output.IsTerminated() {
			return output, nil
		}

		input.Add(t.Header().Name, output)
	}

	return output, nil
}

type SerialBuilder struct {
	task *Serial
}

func NewSerial(name string) *SerialBuilder {
	task := &Serial{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeSerial,
		},
	}
	return &SerialBuilder{task: task}
}

func (b *SerialBuilder) Timeout(timeout time.Duration) *SerialBuilder {
	b.task.Timeout = timeout
	return b
}

func (b *SerialBuilder) Async(async bool) *SerialBuilder {
	b.task.Input.Async = async
	return b
}

func (b *SerialBuilder) Tasks(builders ...orchestrator.Builder) *SerialBuilder {
	var tasks []orchestrator.Task
	for _, builder := range builders {
		tasks = append(tasks, builder.Build())
	}
	b.task.Input.Tasks = tasks
	return b
}

func (b *SerialBuilder) Build() orchestrator.Task {
	return b.task
}
