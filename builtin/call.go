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
	trace := orchestrator.TraceFromContext(ctx).New(c.Name())
	ctx = orchestrator.ContextWithTrace(ctx, trace)

	if err := c.Input.Input.Evaluate(input); err != nil {
		return nil, err
	}

	// Create a new context input since the process will enter a new scope.
	taskInput := orchestrator.NewInput(c.Input.Input.Value)
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

// CallFlow loads the given flow from the given loader, and then execute the flow with the given input.
func CallFlow(ctx context.Context, loader, name string, input map[string]any) (orchestrator.Output, error) {
	def := &orchestrator.TaskDefinition{
		Name: "call",
		Type: TypeCall,
		InputTemplate: orchestrator.InputTemplate{
			"loader": loader,
			"task":   name,
			"input":  input,
		},
	}
	call, err := orchestrator.Construct(orchestrator.NewConstructDecoder(orchestrator.GlobalRegistry), def)
	if err != nil {
		return nil, err
	}
	return call.Execute(ctx, orchestrator.NewInput(nil))
}

// TraceFlow behaves like CallFlow but also enables tracing.
func TraceFlow(ctx context.Context, loader, name string, input map[string]any) orchestrator.Event {
	tr := orchestrator.NewTrace("root")
	ctx = orchestrator.ContextWithTrace(ctx, tr)

	output, err := CallFlow(ctx, loader, name, input)
	tr.AddEvent("call", output, err)

	// To be intuitive, expose the flow's single event from the call trace.
	//
	// root -> call -> flow (serial)
	//  ^       ^        ^
	// tr .Events()[0] .Events[0]
	//
	return tr.Events()[0].Events[0]
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
