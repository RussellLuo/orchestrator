package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/orchestrator"
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
		Constructor: func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			c := &Call{def: def, registry: r}
			if err := r.Decode(def.InputTemplate, &c.Input_); err != nil {
				return nil, err
			}

			if err := c.loadTask(); err != nil {
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
		Raw    bool                              `json:"raw"`
		Input  orchestrator.Expr[map[string]any] `json:"input"`
	}

	// The actual task.
	task     orchestrator.Task
	registry *orchestrator.Registry
}

func NewCall(name string) *Call {
	return &Call{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeCall,
		},
		registry: orchestrator.GlobalRegistry,
	}
}

func (c *Call) Registry(r *orchestrator.Registry) *Call {
	c.registry = r
	return c
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

func (c *Call) Raw() *Call {
	c.Input_.Raw = true
	return c
}

func (c *Call) Input(m map[string]any) *Call {
	c.Input_.Input = orchestrator.Expr[map[string]any]{Expr: m}
	return c
}

func (c *Call) Done() (*Call, error) {
	if err := c.loadTask(); err != nil {
		return nil, err
	}
	return c, nil
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

	inputValue := c.Input_.Input.Expr.(map[string]any)
	if !c.Input_.Raw {
		// If in non-raw mode, the input data will be evaluated.
		var err error
		inputValue, err = c.Input_.Input.EvaluateX(input)
		if err != nil {
			return nil, err
		}
	}

	// Create a new context input since the process will enter a new scope.
	taskInput := orchestrator.NewInput(inputValue)
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

func (c *Call) loadTask() error {
	loader, err := LoaderRegistry.Get(c.Input_.Loader)
	if err != nil {
		return err
	}
	taskDef, err := loader.Load(c.Input_.Task)
	if err != nil {
		return err
	}
	return c.registry.Decode(taskDef, &c.task)
}

// CallFlow loads the given flow from the given loader, and then executes the flow with the given input.
//
// Note that CallFlow is a helper for calling flows which use tasks registered in
// orchestrator.GlobalRegistry. If your case involves tasks registered in a different
// registry, you need to write your own calling code, in which you need to construct
// the call task yourself and specify the registry by using Call.Registry().
func CallFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Output, error) {
	call, err := NewCall("call").Loader(loader).Task(name).Raw().Input(input).Done()
	if err != nil {
		return nil, err
	}
	return call.Execute(ctx, orchestrator.NewInput(nil))
}

// TraceFlow behaves like CallFlow but also enables tracing.
func TraceFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Event, error) {
	call, err := NewCall("call").Loader(loader).Task(name).Raw().Input(input).Done()
	if err != nil {
		return orchestrator.Event{}, err
	}

	event := orchestrator.TraceTask(ctx, call, orchestrator.NewInput(nil))

	// To be intuitive, only expose the flow's single event.
	//
	// call -> flow (serial)
	//  ^       ^
	// event   .Events[0]
	//
	return event.Events[0], nil
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
