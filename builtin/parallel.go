package builtin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeParallel = "parallel"
)

func init() {
	MustRegisterParallel(orchestrator.GlobalRegistry)
}

func MustRegisterParallel(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeParallel,
		New:  func() orchestrator.Task { return new(Parallel) },
	})
}

// Parallel is a composite task that is used to execute its subtasks in parallel.
type Parallel struct {
	orchestrator.TaskHeader

	Input struct {
		Tasks []orchestrator.Task `json:"tasks"`
	} `json:"input"`
}

func (p *Parallel) String() string {
	var inputStrings []string
	for _, t := range p.Input.Tasks {
		inputStrings = append(inputStrings, t.String())
	}
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, tasks:[%s])",
		p.Type,
		p.Name,
		p.Timeout,
		strings.Join(inputStrings, ", "),
	)
}

func (p *Parallel) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	return executeWithTimeout(ctx, input, p.Timeout, p.execute)
}

func (p *Parallel) execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(p.Name)
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	// Scatter
	resultChan := make(chan orchestrator.Result, len(p.Input.Tasks))
	for _, t := range p.Input.Tasks {
		go func(t orchestrator.Task) {
			output, err := t.Execute(ctx, input)
			resultChan <- orchestrator.Result{
				Name:   t.Header().Name,
				Output: output,
				Err:    err,
			}
		}(trace.Wrap(t))
	}

	// Gather
	output := make(map[string]any)
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
		// Sort the original error messages to get a new error with a predictable message.
		sort.Strings(errors)
		return nil, fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return output, nil
}

type ParallelBuilder struct {
	task *Parallel
}

func NewParallel(name string) *ParallelBuilder {
	task := &Parallel{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeParallel,
		},
	}
	return &ParallelBuilder{task: task}
}

func (b *ParallelBuilder) Timeout(timeout time.Duration) *ParallelBuilder {
	b.task.Timeout = timeout
	return b
}

func (b *ParallelBuilder) Tasks(builders ...orchestrator.Builder) *ParallelBuilder {
	var tasks []orchestrator.Task
	for _, builder := range builders {
		tasks = append(tasks, builder.Build())
	}
	b.task.Input.Tasks = tasks
	return b
}

func (b *ParallelBuilder) Build() orchestrator.Task {
	return b.task
}
