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
		New:  func() orchestrator.Task { return new(Call) },
	})
}

// Call is a composite task that is used to call another task with corresponding input.
type Call struct {
	orchestrator.TaskHeader

	Input struct {
		Loader string                            `json:"loader"`
		Task   string                            `json:"task"`
		Raw    bool                              `json:"raw"`
		Input  orchestrator.Expr[map[string]any] `json:"input"`
	} `json:"input"`

	// The actual task.
	task     orchestrator.Task
	registry *orchestrator.Registry
}

func (c *Call) Init(r *orchestrator.Registry) error {
	c.registry = r
	return c.loadTask()
}

func (c *Call) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, task:%s)",
		c.Type,
		c.Name,
		c.Timeout,
		c.Input.Task,
	)
}

func (c *Call) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	trace := orchestrator.TraceFromContext(ctx).New(c.Name)
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	inputValue := c.Input.Input.Expr.(map[string]any)
	if !c.Input.Raw {
		// If in non-raw mode, the input data will be evaluated.
		var err error
		inputValue, err = c.Input.Input.EvaluateX(input)
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
	loader, err := LoaderRegistry.Get(c.Input.Loader)
	if err != nil {
		return err
	}
	taskDef, err := loader.Load(c.Input.Task)
	if err != nil {
		return err
	}
	task, err := c.registry.Construct(taskDef)
	if err != nil {
		return err
	}
	c.task = task
	return nil
}

type CallBuilder struct {
	task *Call
}

func NewCall(name string) *CallBuilder {
	task := &Call{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeCall,
		},
		registry: orchestrator.GlobalRegistry,
	}
	return &CallBuilder{task: task}
}

func (b *CallBuilder) Registry(r *orchestrator.Registry) *CallBuilder {
	b.task.registry = r
	return b
}

func (b *CallBuilder) Timeout(timeout time.Duration) *CallBuilder {
	b.task.Timeout = timeout
	return b
}

func (b *CallBuilder) Loader(name string) *CallBuilder {
	b.task.Input.Loader = name
	return b
}

func (b *CallBuilder) Task(name string) *CallBuilder {
	b.task.Input.Task = name
	return b
}

func (b *CallBuilder) Raw() *CallBuilder {
	b.task.Input.Raw = true
	return b
}

func (b *CallBuilder) Input(m map[string]any) *CallBuilder {
	b.task.Input.Input = orchestrator.Expr[map[string]any]{Expr: m}
	return b
}

func (b *CallBuilder) Build() orchestrator.Task {
	return b.task
}

func (b *CallBuilder) BuildError() (*Call, error) {
	if err := b.task.loadTask(); err != nil {
		return nil, err
	}
	return b.task, nil
}

// CallFlow loads the given flow from the given loader, and then executes the flow with the given input.
//
// Note that CallFlow is a helper for calling flows which use tasks registered in
// orchestrator.GlobalRegistry. If your case involves tasks registered in a different
// registry, you need to write your own calling code, in which you need to construct
// the call task yourself and specify the registry by using Call.Registry().
func CallFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Output, error) {
	call, err := NewCall("call").Loader(loader).Task(name).Raw().Input(input).BuildError()
	if err != nil {
		return nil, err
	}
	return call.Execute(ctx, orchestrator.NewInput(nil))
}

// TraceFlow behaves like CallFlow but also enables tracing.
func TraceFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Event, error) {
	call, err := NewCall("call").Loader(loader).Task(name).Raw().Input(input).BuildError()
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
