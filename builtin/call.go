package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeCall = "call"
)

func init() {
	MustRegisterCall(orchestrator.GlobalRegistry)
}

func MustRegisterCall(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeCall,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			c := &Call{def: def}
			if err := decoder.Decode(def.InputTemplate, &c.Input); err != nil {
				return nil, err
			}

			loader, err := LoaderRegistry.Get(c.Input.Loader)
			if err != nil {
				return nil, err
			}
			taskDef, err := loader.Load(c.Input.Task)
			if err != nil {
				return nil, err
			}
			if err := decoder.Decode(taskDef, &c.task); err != nil {
				return nil, err
			}

			return c, nil
		},
	})
}

// Call is a composite task that is used to call another task with corresponding input.
type Call struct {
	def *orchestrator.TaskDefinition

	Input struct {
		Loader string                            `json:"loader"`
		Task   string                            `json:"task"`
		Input  orchestrator.Expr[map[string]any] `json:"input"`
	}

	// The actual task.
	task orchestrator.Task
}

func NewCall(name string) *Call {
	return &Call{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeCall,
		},
	}
}

func (c *Call) Timeout(timeout time.Duration) *Call {
	c.def.Timeout = timeout
	return c
}

func (c *Call) Loader(name string) *Call {
	c.Input.Loader = name
	return c
}

func (c *Call) Task(name string) *Call {
	c.Input.Task = name
	return c
}

/*
func (c *Call) Input(m map[string]any) *Call {
	return c
}
*/

func (c *Call) Name() string { return c.def.Name }

func (c *Call) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, task:%s)",
		c.def.Type,
		c.def.Name,
		c.def.Timeout,
		c.Input.Task,
	)
}

func (c *Call) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := c.Input.Input.Evaluate(input); err != nil {
		return nil, c.wrapError(err)
	}

	// Create a new context input since the process will enter a new scope.
	taskInput := orchestrator.NewInput(c.Input.Input.Value)
	output, err := c.task.Execute(ctx, taskInput)
	if err != nil {
		return nil, c.wrapError(err)
	}

	// Clear the terminated flag since it only works within the task's scope.
	if output.IsTerminated() {
		output.ClearTerminated()
	}
	return output, nil
}

func (c *Call) wrapError(err error) error {
	return fmt.Errorf("calling task %q: %w", c.task.Name(), err)
}

type Loader interface {
	Load(string) (map[string]any, error)
}

type MapLoader map[string]map[string]any

func (l MapLoader) Load(name string) (map[string]any, error) {
	def, ok := l[name]
	if !ok {
		return nil, fmt.Errorf("task named %q is not found", name)
	}
	return def, nil
}

type loaderRegistry map[string]Loader

func (r loaderRegistry) Get(name string) (Loader, error) {
	loader, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("loader named %q is not found", name)
	}
	return loader, nil
}

func (r loaderRegistry) MustRegister(name string, loader Loader) {
	if _, ok := r[name]; ok {
		panic(fmt.Errorf("loader named %q is already registered", name))
	}
	r[name] = loader
}

var LoaderRegistry = loaderRegistry{}
