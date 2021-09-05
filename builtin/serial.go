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

type Result struct {
	Name   string
	Output orchestrator.Output
	Err    error
}

func executeWithTimeout(ctx context.Context, decoder *orchestrator.Decoder, timeout time.Duration, f func(context.Context, *orchestrator.Decoder) (orchestrator.Output, error)) (orchestrator.Output, error) {
	if timeout <= 0 {
		// Execute f directly.
		return f(ctx, decoder)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan Result, 1)
	go func() {
		output, err := f(ctx, decoder)
		resultChan <- Result{Output: output, Err: err}
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
		Tasks []orchestrator.Task `orchestrator:"tasks"`
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

func (s *Serial) InputString() string {
	var inputStrings []string
	for _, t := range s.Input.Tasks {
		inputStrings = append(inputStrings, t.InputString())
	}
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, tasks:[%s])",
		s.def.Type,
		s.def.Name,
		s.def.Timeout,
		strings.Join(inputStrings, ", "),
	)
}

func (s *Serial) Definition() *orchestrator.TaskDefinition {
	return s.def
}

func (s *Serial) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	return executeWithTimeout(ctx, decoder, s.def.Timeout, s.execute)
}

func (s *Serial) execute(ctx context.Context, decoder *orchestrator.Decoder) (output orchestrator.Output, err error) {
	for _, t := range s.Input.Tasks {
		output, err = t.Execute(ctx, decoder)
		if err != nil {
			return nil, err
		}

		if output.IsTerminated() {
			return output, nil
		}

		decoder.AddOutput(t.Definition().Name, output)
	}
	return output, nil
}
