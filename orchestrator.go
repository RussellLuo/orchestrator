package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RussellLuo/structool"
	"github.com/xeipuuv/gojsonschema"
)

type Input struct {
	*Evaluator
}

func NewInput(input map[string]any) Input {
	evaluator := NewEvaluator()
	evaluator.Add("input", input)
	return Input{Evaluator: evaluator}
}

type Output map[string]any

func (o Output) SetTerminated() {
	o["terminated"] = true
}

func (o Output) ClearTerminated() {
	delete(o, "terminated")
}

func (o Output) IsTerminated() bool {
	terminated, ok := o["terminated"].(bool)
	return ok && terminated
}

func (o Output) Iterator() (iterator *Iterator, ok bool) {
	iterator, ok = o["iterator"].(*Iterator)
	return
}

func (o Output) Actor() (actor *Actor, ok bool) {
	actor, ok = o["actor"].(*Actor)
	return
}

type Schema struct {
	Input  map[string]any `json:"input"`
	Output map[string]any `json:"output"`
}

func (s Schema) Validate(input map[string]any) error {
	if len(s.Input) == 0 {
		// No input schema specified, do no validation.
		return nil
	}

	schemaLoader := gojsonschema.NewGoLoader(s.Input)
	inputLoader := gojsonschema.NewGoLoader(input)

	result, err := gojsonschema.Validate(schemaLoader, inputLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

type TaskHeader struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	//Schema        Schema        `json:"schema"`
	Timeout time.Duration `json:"timeout"`
}

func (h TaskHeader) Header() TaskHeader { return h }

type Initializer interface {
	// Init initializes an application with the given context ctx.
	// It will return an error if it fails.
	Init(r *Registry) error
}

type Task interface {
	// Header returns the header fields of the task.
	Header() TaskHeader

	// String returns a string representation of the task.
	String() string

	// Execute executes the task with the given input.
	Execute(context.Context, Input) (Output, error)
}

type TaskFactory struct {
	Type string
	New  func() Task
}

type Registry struct {
	factories map[string]*TaskFactory
	decoder   *structool.Codec
}

func NewRegistry() *Registry {
	r := new(Registry)
	r.factories = make(map[string]*TaskFactory)
	r.decoder = structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
		decodeDefinitionToTask(r),
	)
	return r
}

func (r *Registry) Register(factory *TaskFactory) error {
	if _, ok := r.factories[factory.Type]; ok {
		return fmt.Errorf("factory for task type %q is already registered", factory.Type)
	}

	r.factories[factory.Type] = factory
	return nil
}

// MustRegister is like Register but panics if there is an error.
func (r *Registry) MustRegister(factory *TaskFactory) {
	if err := r.Register(factory); err != nil {
		panic(err)
	}
}

func (r *Registry) Construct(m map[string]any) (Task, error) {
	typ := ""
	if v, ok := m["type"]; ok {
		if s, ok := v.(string); ok {
			typ = s
		}
	}
	factory, ok := r.factories[typ]
	if !ok {
		return nil, fmt.Errorf("factory for task type %q is not found", typ)
	}

	task := factory.New()
	if err := r.decoder.Decode(m, task); err != nil {
		return nil, err
	}

	if initializer, ok := task.(Initializer); ok {
		if err := initializer.Init(r); err != nil {
			return nil, err
		}
	}
	return task, nil
}

func (r *Registry) ConstructFromJSON(data []byte) (Task, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return r.Construct(m)
}

func MustRegister(factory *TaskFactory) {
	GlobalRegistry.MustRegister(factory)
}

func Construct(m map[string]any) (Task, error) {
	return GlobalRegistry.Construct(m)
}

func ConstructFromJSON(data []byte) (Task, error) {
	return GlobalRegistry.ConstructFromJSON(data)
}

var GlobalRegistry = NewRegistry()
