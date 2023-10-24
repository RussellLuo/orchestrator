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

func MustRegisterCall(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeCall,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			c := &Call{def: def}
			if err := decoder.Decode(def.InputTemplate, &c.Input_); err != nil {
				return nil, err
			}

			loader, err := LoaderRegistry.Get(c.Input_.Loader)
			if err != nil {
				return nil, err
			}
			taskDef, err := loader.Load(c.Input_.Task)
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

	Input_ struct {
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
	c.Input_.Loader = name
	return c
}

func (c *Call) Task(name string) *Call {
	c.Input_.Task = name
	return c
}

func (c *Call) Input(m map[string]any) *Call {
	c.Input_.Input = orchestrator.Expr[map[string]any]{Expr: m}
	return c
}

func (c *Call) Name() string { return c.def.Name }

func (c *Call) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, task:%s)",
		c.def.Type,
		c.def.Name,
		c.def.Timeout,
		c.Input_.Task,
	)
}

func (c *Call) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(c.Name())
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	if err := c.Input_.Input.Evaluate(input); err != nil {
		return nil, err
	}

	// Create a new context input since the process will enter a new scope.
	taskInput := orchestrator.NewInput(c.Input_.Input.Value)
	output, err := trace.Wrap(c.task).Execute(ctx, taskInput)
	if err != nil {
		return nil, err
	}

	// Clear the terminated flag since it only works within the task's scope.
	if output.IsTerminated() {
		output.ClearTerminated()
	}
	return output, nil
}

// CallFlow loads the given flow from the given loader, and then executes the flow with the given input.
func CallFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Output, error) {
	call := NewCall("call").Loader(loader).Task(name).Input(input)
	return call.Execute(ctx, orchestrator.NewInput(nil))
}

// TraceFlow behaves like CallFlow but also enables tracing.
func TraceFlow(ctx context.Context, loader, name string, input map[string]any) orchestrator.Event {
	call := NewCall("call").Loader(loader).Task(name).Input(input)
	event := orchestrator.TraceTask(ctx, call, orchestrator.NewInput(nil))

	// To be intuitive, only expose the flow's single event.
	//
	// call -> flow (serial)
	//  ^       ^
	// event   .Events[0]
	//
	return event.Events[0]
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
