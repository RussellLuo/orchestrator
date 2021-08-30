package builtin

import (
	"context"
	"fmt"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeDecision = "decision"
)

// Decision is a composite task that is is similar to the `switch` statement in Go.
type Decision struct {
	*orchestrator.TaskDefinition

	input struct {
		Switch  interface{}                       `orchestrator:"switch"`
		Cases   map[interface{}]orchestrator.Task `orchestrator:"cases"`
		Default orchestrator.Task                 `orchestrator:"default"`
	}
}

func NewDecision(o *orchestrator.Orchestrator, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeDecision {
		def.Type = TypeDecision
	}

	d := &Decision{TaskDefinition: def}

	var input struct {
		Switch  interface{}                                  `orchestrator:"switch"`
		Cases   map[interface{}]*orchestrator.TaskDefinition `orchestrator:"cases"`
		Default *orchestrator.TaskDefinition                 `orchestrator:"default"`
	}
	decoder := orchestrator.NewDecoder().NoRendering()
	if err := decoder.Decode(def.InputTemplate, &input); err != nil {
		return nil, err
	}

	d.input.Switch = input.Switch

	// Build cases
	d.input.Cases = make(map[interface{}]orchestrator.Task)
	names := make(map[string]bool) // Detect duplicate task name.
	for v, td := range input.Cases {
		if _, ok := names[td.Name]; ok {
			return nil, fmt.Errorf("duplicate task name %q", td.Name)
		}
		names[td.Name] = true

		task, err := o.Construct(td)
		if err != nil {
			return nil, err
		}
		d.input.Cases[v] = task
	}

	// Build default
	if input.Default != nil {
		task, err := o.Construct(input.Default)
		if err != nil {
			return nil, err
		}
		d.input.Default = task
	}

	return d, nil
}

func (d *Decision) Definition() *orchestrator.TaskDefinition {
	return d.TaskDefinition
}

func (d *Decision) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	var switchValue interface{}
	if err := decoder.Decode(d.input.Switch, &switchValue); err != nil {
		return nil, err
	}

	task, ok := d.input.Cases[switchValue]
	if !ok {
		if d.input.Default != nil {
			return d.input.Default.Execute(ctx, decoder)
		}
		return nil, nil
	}

	return task.Execute(ctx, decoder)
}
