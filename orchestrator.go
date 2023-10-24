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

type InputTemplate map[string]any

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

type TaskDefinition struct {
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Description   string        `json:"description"`
	Schema        Schema        `json:"schema"`
	Timeout       time.Duration `json:"timeout"`
	InputTemplate InputTemplate `json:"input"`
}

type Task interface {
	// Name returns the name of the task.
	Name() string

	// String returns a string representation of the task.
	String() string

	// Execute executes the task with the given input.
	Execute(context.Context, Input) (Output, error)
}

type TaskFactory struct {
	Type        string
	Constructor func(*structool.Codec, *TaskDefinition) (Task, error)
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

func (r *Registry) Construct(def *TaskDefinition) (Task, error) {
	factory, ok := r.factories[def.Type]
	if !ok {
		return nil, fmt.Errorf("factory for task type %q is not found", def.Type)
	}
	return factory.Constructor(r.decoder, def)
}

func (r *Registry) ConstructFromMap(m map[string]any) (Task, error) {
	codec := structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
	)
	var def *TaskDefinition
	if err := codec.Decode(m, &def); err != nil {
		return nil, err
	}

	return r.Construct(def)
}

func (r *Registry) ConstructFromJSON(data []byte) (Task, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return r.ConstructFromMap(m)
}

func MustRegister(factory *TaskFactory) {
	GlobalRegistry.MustRegister(factory)
}

func Construct(def *TaskDefinition) (Task, error) {
	return GlobalRegistry.Construct(def)
}

func ConstructFromMap(m map[string]any) (Task, error) {
	return GlobalRegistry.ConstructFromMap(m)
}

func ConstructFromJSON(data []byte) (Task, error) {
	return GlobalRegistry.ConstructFromJSON(data)
}

var GlobalRegistry = NewRegistry()
