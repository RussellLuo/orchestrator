package builtin

import (
	"context"
	"fmt"
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

func parseInputTasks(o *orchestrator.Orchestrator, inputTemplate orchestrator.InputTemplate, tasks *[]orchestrator.Task) error {
	var input struct {
		Tasks []*orchestrator.TaskDefinition `orchestrator:"tasks"`
	}
	decoder := orchestrator.NewDecoder().NoRendering()
	if err := decoder.Decode(inputTemplate, &input); err != nil {
		return err
	}

	names := make(map[string]bool) // Detect duplicate task name.
	for _, d := range input.Tasks {
		if _, ok := names[d.Name]; ok {
			return fmt.Errorf("duplicate task name %q", d.Name)
		}
		names[d.Name] = true

		task, err := o.Construct(d)
		if err != nil {
			return err
		}
		*tasks = append(*tasks, task)
	}

	return nil
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
	*orchestrator.TaskDefinition

	tasks []orchestrator.Task
}

func NewSerial(o *orchestrator.Orchestrator, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeSerial {
		def.Type = TypeSerial
	}

	s := &Serial{TaskDefinition: def}
	if err := parseInputTasks(o, def.InputTemplate, &s.tasks); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Serial) Definition() *orchestrator.TaskDefinition {
	return s.TaskDefinition
}

func (s *Serial) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	return executeWithTimeout(ctx, decoder, s.Timeout, s.execute)
}

func (s *Serial) execute(ctx context.Context, decoder *orchestrator.Decoder) (output orchestrator.Output, err error) {
	for _, t := range s.tasks {
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
