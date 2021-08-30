package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeParallel = "parallel"
)

// Parallel is a composite task that is used to execute its subtasks in parallel.
type Parallel struct {
	*orchestrator.TaskDefinition

	tasks []orchestrator.Task
}

func NewParallel(o *orchestrator.Orchestrator, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeParallel {
		def.Type = TypeParallel
	}

	p := &Parallel{TaskDefinition: def}
	if err := parseInputTasks(o, def.InputTemplate, &p.tasks); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Parallel) Definition() *orchestrator.TaskDefinition {
	return p.TaskDefinition
}

func (p *Parallel) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	return executeWithTimeout(ctx, decoder, p.Timeout, p.execute)
}

func (p *Parallel) execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	// Scatter
	resultChan := make(chan Result, len(p.tasks))
	for _, t := range p.tasks {
		go func(t orchestrator.Task) {
			output, err := t.Execute(ctx, decoder)
			resultChan <- Result{
				Name:   t.Definition().Name,
				Output: output,
				Err:    err,
			}
		}(t)
	}

	// Gather
	output := make(map[string]interface{})
	var errors []string
	for i := 0; i < cap(resultChan); i++ {
		result := <-resultChan
		if result.Err != nil {
			errors = append(errors, result.Err.Error())
		} else {
			output[result.Name] = result.Output
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return output, nil
}
