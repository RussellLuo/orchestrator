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

type Validator interface {
	// Validate validates the given input against the task's schema.
	Validate(map[string]any) error
}

type TaskFactory struct {
	Type        string
	Constructor func(*structool.Codec, *TaskDefinition) (Task, error)
}

type Registry map[string]*TaskFactory

func (r Registry) Register(factory *TaskFactory) error {
	if _, ok := r[factory.Type]; ok {
		return fmt.Errorf("factory for task type %q is already registered", factory.Type)
	}

	r[factory.Type] = factory
	return nil
}

// MustRegister is like Register but panics if there is an error.
func (r Registry) MustRegister(factory *TaskFactory) {
	if err := r.Register(factory); err != nil {
		panic(err)
	}
}

func (r Registry) Construct(decoder *structool.Codec, def *TaskDefinition) (Task, error) {
	factory, ok := r[def.Type]
	if !ok {
		return nil, fmt.Errorf("factory for task type %q is not found", def.Type)
	}
	return factory.Constructor(decoder, def)
}

func (r Registry) ConstructFromMap(decoder *structool.Codec, m map[string]any) (Task, error) {
	codec := structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
	)
	var def *TaskDefinition
	if err := codec.Decode(m, &def); err != nil {
		return nil, err
	}

	return r.Construct(decoder, def)
}

func (r Registry) ConstructFromJSON(decoder *structool.Codec, data []byte) (Task, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return r.ConstructFromMap(decoder, m)
}

func MustRegister(factory *TaskFactory) {
	GlobalRegistry.MustRegister(factory)
}

func Construct(decoder *structool.Codec, def *TaskDefinition) (Task, error) {
	return GlobalRegistry.Construct(decoder, def)
}

func ConstructFromMap(decoder *structool.Codec, m map[string]any) (Task, error) {
	return GlobalRegistry.ConstructFromMap(decoder, m)
}

func ConstructFromJSON(decoder *structool.Codec, data []byte) (Task, error) {
	return GlobalRegistry.ConstructFromJSON(decoder, data)
}

var GlobalRegistry = Registry{}
