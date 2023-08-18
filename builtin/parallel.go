package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeParallel = "parallel"
)

func init() {
	MustRegisterParallel(orchestrator.GlobalRegistry)
}

func MustRegisterParallel(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeParallel,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			p := &Parallel{def: def}
			if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
				return nil, err
			}
			return p, nil
		},
	})
}

// Parallel is a composite task that is used to execute its subtasks in parallel.
type Parallel struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Tasks []orchestrator.Task `orchestrator:"tasks"`
	}
}

func NewParallel(name string) *Parallel {
	return &Parallel{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeParallel,
		},
	}
}

func (p *Parallel) Timeout(timeout time.Duration) *Parallel {
	p.def.Timeout = timeout
	return p
}

func (p *Parallel) Tasks(tasks ...orchestrator.Task) *Parallel {
	p.Input.Tasks = tasks
	return p
}

func (p *Parallel) InputString() string {
	var inputStrings []string
	for _, t := range p.Input.Tasks {
		inputStrings = append(inputStrings, t.InputString())
	}
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, tasks:[%s])",
		p.def.Type,
		p.def.Name,
		p.def.Timeout,
		strings.Join(inputStrings, ", "),
	)
}

func (p *Parallel) Definition() *orchestrator.TaskDefinition {
	return p.def
}

func (p *Parallel) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	return executeWithTimeout(ctx, decoder, p.def.Timeout, p.execute)
}

func (p *Parallel) execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	// Scatter
	resultChan := make(chan Result, len(p.Input.Tasks))
	for _, t := range p.Input.Tasks {
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
